package deleteprocess

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

type DeleteProcessArgs struct {
	OutputFormat string `json:\"output_format,omitempty\"` // OutputFormat specifies the desired output format (e.g. \"json\"). Defaults to text.
	PID          int    `json:"pid"`
	Signal       string `json:"signal"`
	Privileged   bool   `json:"privileged"`
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
