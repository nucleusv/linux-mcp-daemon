package deleteprocess

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// DeleteProcessArgs defines the parameters for the processes/delete tool.
type DeleteProcessArgs struct {
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
	PID          int    `json:"pid"`                     // PID is the Process ID to terminate. Required.
	Signal       string `json:"signal,omitempty"`        // Signal is the signal to send (e.g., "SIGTERM", "SIGKILL"). Defaults to "SIGTERM".
	Privileged   bool   `json:"privileged,omitempty"`    // Privileged executes the kill command as root.
}

var validSignals = map[string]bool{
	"SIGTERM": true, "SIGKILL": true, "SIGHUP": true, "SIGINT": true, "SIGQUIT": true,
	"SIGUSR1": true, "SIGUSR2": true, "SIGSTOP": true, "SIGCONT": true, "SIGABRT": true,
}

func Delete(argsJSON []byte) (string, error) {
	var args DeleteProcessArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	if args.PID <= 0 {
		return "", fmt.Errorf("invalid PID specified")
	}

	// Never signal init or this daemon itself (the worker's parent is the
	// mcpd master): killing either takes the whole host or the daemon down,
	// which no "terminate a process" request should be able to do.
	if args.PID == 1 || args.PID == os.Getppid() || args.PID == os.Getpid() {
		return "", fmt.Errorf("refusing to signal protected PID %d (init or the mcpd daemon itself)", args.PID)
	}

	signal := "SIGTERM"
	if args.Signal != "" {
		s := strings.ToUpper(strings.TrimSpace(args.Signal))
		if !strings.HasPrefix(s, "SIG") {
			s = "SIG" + s
		}
		if !validSignals[s] {
			return "", fmt.Errorf("invalid signal %q (e.g. SIGTERM, SIGKILL, SIGHUP, SIGINT)", args.Signal)
		}
		signal = s
	}

	cmd := exec.Command("kill", "-s", strings.TrimPrefix(signal, "SIG"), "--", strconv.Itoa(args.PID))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("kill error: %v, stderr: %s", err, stderr.String())
	}

	return fmt.Sprintf("Successfully sent signal %s to process %d", signal, args.PID), nil
}
