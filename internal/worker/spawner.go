package worker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
)

// Containerized indicates this daemon process itself runs inside a
// container with its own private root filesystem (e.g. Kubernetes), as
// opposed to running directly on the host with no container boundary. Set
// once at daemon startup from configs/daemon.yaml's worker.containerized
// field. When true, every privileged worker call also joins the real host's
// mount namespace (see JoinHostMountNamespace) - there is no separate,
// weaker "root inside the container only" tier, since nothing in this
// daemon's actual usage wants that distinction: a privileged caller asking
// for root always means root on the real host being administered.
var Containerized bool

// NoRoot is set by `mcpd stdio`: nothing may run as root, whatever the
// caller asks for. Over stdio there is no token and no MCP user, so a root
// grant would belong to whoever controls the client's config. Checked here,
// the one place every tool and resource call goes through.
var NoRoot bool

// ErrNoRoot is the refusal of privileged: true under NoRoot.
var ErrNoRoot = errors.New("privileged: true is not available when mcpd serves over stdio - it never runs anything as root. For root on specific tools, use the mcpd network daemon with a grant in mcp-sudo.yaml")

// SpawnWorker forks a child process to execute the requested tool with strict privilege isolation.
func SpawnWorker(username, toolName string, toolArgs []byte, privileged bool, sudoCfg *config.SudoConfig, timeoutSeconds int) (string, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return "", fmt.Errorf("failed to lookup OS user %s: %v", username, err)
	}

	uid, _ := strconv.Atoi(u.Uid)
	// An MCP user must be an unprivileged OS account. With uid 0 every call
	// would run as root - privileged: true or not - past every limit in
	// mcp-sudo.yaml (paths, network, sysctl).
	if uid == 0 {
		return "", fmt.Errorf("MCP user %q is the OS account %s with uid 0 (root) - refused: map MCP users to unprivileged accounts, and grant root per tool in mcp-sudo.yaml", username, u.Username)
	}
	targetUID := uint32(uid)
	gid, _ := strconv.Atoi(u.Gid)
	targetGID := uint32(gid)
	// Supplementary groups too - without them the worker would keep none
	// (or, worse, whatever the root master holds).
	var targetGroups []uint32
	if ids, err := u.GroupIds(); err == nil {
		for _, id := range ids {
			if n, err := strconv.Atoi(id); err == nil {
				targetGroups = append(targetGroups, uint32(n))
			}
		}
	}

	if privileged && NoRoot {
		return "", ErrNoRoot
	}
	if privileged {
		if strings.HasPrefix(toolName, "read_") {
			targetUID, targetGID, targetGroups = 0, 0, []uint32{0} // Internal resource workers (caller already verified)
		} else if sudoCfg != nil && sudoCfg.CanRunAsRoot(username, toolName) {
			targetUID, targetGID, targetGroups = 0, 0, []uint32{0} // Elevate to root
		} else {
			return "", fmt.Errorf("Permission denied. Hint: You are not authorized to use 'privileged: true' for this tool in mcp-sudo.yaml")
		}
	}

	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to determine executable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	// Arguments go over stdin, never argv: argv is world-readable via
	// /proc/<pid>/cmdline and ps, and tool arguments carry file contents,
	// HTTP Authorization headers and the like.
	cmd := exec.CommandContext(ctx, exe, "worker", toolName)
	cmd.Stdin = bytes.NewReader(toolArgs)
	if privileged && Containerized {
		cmd.Env = append(os.Environ(), "MCPD_HOST_ROOT=1")
	}

	// Enforce strict UID *and GID* isolation. Setting only Uid used to leave
	// Gid at its zero value - so every "unprivileged" worker ran with group
	// root (gid 0) and could use any group-root file permission.
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	if euid := os.Geteuid(); euid != 0 {
		// A non-root mcpd (`mcpd stdio` started by an ordinary user) can't
		// switch accounts - setgroups needs CAP_SETGID. Its workers simply
		// stay the account it runs as, which must be the caller.
		if targetUID != uint32(euid) {
			return "", fmt.Errorf("mcpd runs as uid %d and cannot start a worker as %s (uid %d)", euid, username, targetUID)
		}
	} else {
		cmd.SysProcAttr.Credential = &syscall.Credential{Uid: targetUID, Gid: targetGID, Groups: targetGroups}
	}

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("timed out after %d seconds - the call was stopped", timeoutSeconds)
	}
	if err != nil {
		stderr := strings.TrimSpace(errBuf.String())
		var exitErr *exec.ExitError
		// Exit status 1 with a message is a tool's own, ordinary failure
		// ("no such file", "invalid mode"): pass the message on as it is.
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 && stderr != "" {
			if !privileged && looksLikePermissionError(stderr) && sudoCfg != nil && sudoCfg.CanRunAsRoot(username, toolName) {
				stderr += fmt.Sprintf(" (%s is granted to you as root in mcp-sudo.yaml: retry with privileged: true)", toolName)
			}
			return "", errors.New(stderr)
		}
		// Anything else - killed by a signal, a crash - is the worker failing.
		if stderr != "" {
			return "", fmt.Errorf("worker failed: %v: %s", err, stderr)
		}
		return "", fmt.Errorf("worker failed: %v", err)
	}

	return outBuf.String(), nil
}

// looksLikePermissionError reports whether a tool's error message is the
// kind running as root would fix.
func looksLikePermissionError(msg string) bool {
	m := strings.ToLower(msg)
	for _, s := range []string{"permission denied", "operation not permitted", "insufficient permissions", "not permitted to"} {
		if strings.Contains(m, s) {
			return true
		}
	}
	return false
}
