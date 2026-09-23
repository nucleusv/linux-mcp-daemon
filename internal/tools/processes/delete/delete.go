package deleteprocess

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"
)

// DeleteProcessArgs defines the parameters for the processes/delete tool.
type DeleteProcessArgs struct {
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
	PID          int    `json:"pid"`                     // PID is the Process ID to terminate. Required.
	Signal       string `json:"signal,omitempty"`        // Signal is the signal to send (e.g., "SIGTERM", "SIGKILL"). Defaults to "SIGTERM".
	Privileged   bool   `json:"privileged,omitempty"`    // Privileged executes the kill command as root.
}

// validSignals are the signals this tool may send, by name.
var validSignals = map[string]syscall.Signal{
	"SIGTERM": syscall.SIGTERM, "SIGKILL": syscall.SIGKILL, "SIGHUP": syscall.SIGHUP,
	"SIGINT": syscall.SIGINT, "SIGQUIT": syscall.SIGQUIT, "SIGUSR1": syscall.SIGUSR1,
	"SIGUSR2": syscall.SIGUSR2, "SIGSTOP": syscall.SIGSTOP, "SIGCONT": syscall.SIGCONT,
	"SIGABRT": syscall.SIGABRT,
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
		if _, ok := validSignals[s]; !ok {
			return "", fmt.Errorf("invalid signal %q (e.g. SIGTERM, SIGKILL, SIGHUP, SIGINT)", args.Signal)
		}
		signal = s
	}

	// kill(2) directly rather than the kill binary. PID is already > 0 here,
	// so this can never be the kill(-1)/kill(0) broadcast forms.
	if err := syscall.Kill(args.PID, validSignals[signal]); err != nil {
		switch err {
		case syscall.ESRCH:
			return "", fmt.Errorf("no such process: %d", args.PID)
		case syscall.EPERM:
			return "", fmt.Errorf("not permitted to signal process %d (owned by another user - retry with privileged: true if authorized)", args.PID)
		}
		return "", fmt.Errorf("failed to send %s to process %d: %v", signal, args.PID, err)
	}

	return fmt.Sprintf("Successfully sent signal %s to process %d", signal, args.PID), nil
}
