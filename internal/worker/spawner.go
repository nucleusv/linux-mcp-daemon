package worker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
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

// SpawnWorker forks a child process to execute the requested tool with strict privilege isolation.
func SpawnWorker(username, toolName string, toolArgs []byte, privileged bool, sudoCfg *config.SudoConfig, timeoutSeconds int) (string, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return "", fmt.Errorf("failed to lookup OS user %s: %v", username, err)
	}

	uid, _ := strconv.Atoi(u.Uid)
	targetUID := uint32(uid)

	if privileged {
		if strings.HasPrefix(toolName, "read_") {
			targetUID = 0 // Internal resource workers (caller already verified)
		} else if sudoCfg != nil && sudoCfg.CanRunAsRoot(username, toolName) {
			targetUID = 0 // Elevate to root
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

	cmd := exec.CommandContext(ctx, exe, "worker", toolName, string(toolArgs))
	if privileged && Containerized {
		cmd.Env = append(os.Environ(), "MCPD_HOST_ROOT=1")
	}

	// Enforce strict UID isolation.
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: targetUID}

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("worker execution failed: timeout exceeded (%d seconds). Process was forcefully terminated.", timeoutSeconds)
	}
	if err != nil {
		// Include stderr in the error response if it fails
		if errBuf.Len() > 0 {
			return "", fmt.Errorf("worker execution failed: %v. Stderr: %s", err, errBuf.String())
		}
		
		// If it's a standard exit error, provide a helpful hint if they weren't using privileges
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() != 0 && !privileged {
			if sudoCfg != nil && sudoCfg.CanRunAsRoot(username, toolName) {
				return "", fmt.Errorf("worker failed: %v. Hint: You are authorized to run this tool as root. Try again with 'privileged: true'. Output: %s", err, outBuf.String())
			}
		}
		
		return "", fmt.Errorf("worker execution failed: %v. Output: %s", err, outBuf.String())
	}

	return outBuf.String(), nil
}
