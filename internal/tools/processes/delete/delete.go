package deleteprocess

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// DeleteProcessArgs defines the parameters for the processes/delete tool.
type DeleteProcessArgs struct {
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
	PID          int    `json:"pid"`                     // PID is the Process ID to terminate. Required.
	Signal       string `json:"signal,omitempty"`        // Signal is the signal to send (e.g., "SIGTERM", "SIGKILL"). Defaults to "SIGTERM".
	Privileged   bool   `json:"privileged,omitempty"`    // Privileged executes the kill command as root.
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

	signal := "SIGTERM"
	if args.Signal != "" {
		signal = args.Signal
	}

	cmd := exec.Command("kill", "-s", signal, strconv.Itoa(args.PID))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("kill error: %v, stderr: %s", err, stderr.String())
	}

	return fmt.Sprintf("Successfully sent signal %s to process %d", signal, args.PID), nil
}
