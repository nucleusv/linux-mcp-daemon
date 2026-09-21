package connections

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type GetConnectionsArgs struct {
	OutputFormat string `json:\"output_format,omitempty\"` // OutputFormat specifies the desired output format (e.g. \"json\"). Defaults to text.
	State      string `json:"state"`
	Port       int    `json:"port"`
	Privileged bool   `json:"privileged"`
}

func GetConnections(argsJSON []byte) (string, error) {
	var args GetConnectionsArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	cmdArgs := []string{"-tulnp"} // TCP, UDP, listening, numeric, show processes
	if args.State != "" {
		cmdArgs = append(cmdArgs, "state", args.State)
	}
	if args.Port > 0 {
		cmdArgs = append(cmdArgs, fmt.Sprintf("( sport = :%d or dport = :%d )", args.Port, args.Port))
	}

	cmd := exec.Command("ss", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ss error: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
