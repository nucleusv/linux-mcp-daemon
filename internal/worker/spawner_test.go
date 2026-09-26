package worker

import (
	"os"
	"os/user"
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
)

func TestLooksLikePermissionError(t *testing.T) {
	for msg, want := range map[string]bool{
		"failed to open file: open /etc/shadow: permission denied":                    true,
		"/tmp/x: operation not permitted":                                             true,
		"No journal files were opened due to insufficient permissions.":               true,
		"not permitted to signal process 66594 (owned by another user)":               true,
		"failed to open file: open /tmp/x: no such file or directory":                 false,
		`invalid mode "999": expected octal (0755) or symbolic ([ugoa][+-=][rwxXst])`: false,
	} {
		if got := looksLikePermissionError(msg); got != want {
			t.Errorf("%q: got %t", msg, got)
		}
	}
}

func TestSpawnWorkerRefusesRootAccount(t *testing.T) {
	_, err := SpawnWorker("root", "system/os-release", []byte("{}"), false, nil, 5)
	if err == nil || !strings.Contains(err.Error(), "uid 0") {
		t.Fatalf("want a uid 0 refusal, got %v", err)
	}
}

func TestSpawnWorkerNoRootRefusesPrivileged(t *testing.T) {
	NoRoot = true
	defer func() { NoRoot = false }()
	u, err := user.Current()
	if err != nil || u.Uid == "0" {
		t.Skip("needs a non-root current user")
	}
	// Even an internal resource worker (read_*), which the network daemon
	// elevates without a tool grant, and even with a grant in the config.
	sudo := &config.SudoConfig{Users: map[string]config.UserSudo{u.Username: {Privileged: config.PrivilegedConfig{Tools: map[string]config.ToolPrivilege{"system/os-release": {Allowed: true}}}}}}
	for _, tool := range []string{"system/os-release", "read_usb"} {
		_, err := SpawnWorker(u.Username, tool, []byte("{}"), true, sudo, 5)
		if err == nil || !strings.Contains(err.Error(), "stdio") {
			t.Errorf("%s with privileged: true under NoRoot: want the stdio refusal, got %v", tool, err)
		}
	}
}

func TestSpawnWorkerAsSelfWithoutRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("run as a non-root user")
	}

	// A non-root mcpd can't start a worker as someone else.
	if _, err := SpawnWorker("nobody", "system/os-release", []byte("{}"), false, nil, 5); err == nil || !strings.Contains(err.Error(), "cannot start a worker") {
		t.Errorf("worker as another user from a non-root mcpd: want a refusal, got %v", err)
	}

}
