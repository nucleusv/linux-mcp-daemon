package journal_control

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// JournalControlArgs defines the parameters for the logs/journalctl tool.
type JournalControlArgs struct {
	Unit         string `json:"unit,omitempty"`          // Unit filters by systemd unit (e.g., "kubelet.service").
	Lines        int    `json:"lines,omitempty"`         // Lines specifies the number of most recent lines to return. Defaults to 100.
	Since        string `json:"since,omitempty"`         // Since filters logs on or newer than the specified date/time (e.g., "1 hour ago", "yesterday").
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
}

func JournalControl(argsJSON []byte) (string, error) {
	var args JournalControlArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	lines := args.Lines
	if lines <= 0 {
		lines = 100
	}

	cmdArgs := []string{"-n", fmt.Sprintf("%d", lines), "--no-pager"}

	if args.Unit != "" {
		cmdArgs = append(cmdArgs, "-u", args.Unit)
	}

	if args.Since != "" {
		cmdArgs = append(cmdArgs, "--since", args.Since)
	}

	if args.OutputFormat == "json" {
		cmdArgs = append(cmdArgs, "-o", "json")
	}

	cmd := exec.Command("journalctl", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("journalctl error: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
