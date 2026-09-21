package processes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type GetProcessesArgs struct {
	OutputFormat string `json:\"output_format,omitempty\"` // OutputFormat specifies the desired output format (e.g. \"json\"). Defaults to text.
	User       string `json:"user"`
	SortBy     string `json:"sort_by"`
	Limit      int    `json:"limit"`
	Privileged bool   `json:"privileged"`
}

func GetProcesses(argsJSON []byte) (string, error) {
	var args GetProcessesArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	cmdArgs := []string{"aux"}
	if args.User != "" {
		cmdArgs = []string{"-U", args.User, "-u", args.User, "u"}
	}

	if args.SortBy != "" {
		switch args.SortBy {
		case "cpu":
			cmdArgs = append(cmdArgs, "--sort=-%cpu")
		case "mem":
			cmdArgs = append(cmdArgs, "--sort=-%mem")
		case "pid":
			cmdArgs = append(cmdArgs, "--sort=pid")
		}
	}

	cmd := exec.Command("ps", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ps error: %v, stderr: %s", err, stderr.String())
	}

	out := stdout.String()

	// Handle limit if specified (approximate by lines)
	if args.Limit > 0 {
		var limited bytes.Buffer
		lines := 0
		for _, c := range []byte(out) {
			limited.WriteByte(c)
			if c == '\n' {
				lines++
				if lines > args.Limit {
					break
				}
			}
		}
		out = limited.String()
	}

	return out, nil
}
