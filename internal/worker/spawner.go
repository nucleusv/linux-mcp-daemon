package worker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
)

// SpawnWorker forks a child process to execute the requested tool with strict privilege isolation.
func SpawnWorker(username, toolName string, toolArgs []byte, privileged bool, sudoCfg *config.SudoConfig, timeoutSeconds int) (string, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return "", fmt.Errorf("failed to lookup OS user %s: %v", username, err)
	}

	uid, _ := strconv.Atoi(u.Uid)
	targetUID := uint32(uid)

	if privileged {
		if sudoCfg != nil && sudoCfg.CanRunAsRoot(username, toolName) {
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
