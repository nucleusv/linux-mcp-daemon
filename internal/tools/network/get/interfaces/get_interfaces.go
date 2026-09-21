package interfaces

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type GetInterfacesArgs struct {
	OutputFormat string `json:\"output_format,omitempty\"` // OutputFormat specifies the desired output format (e.g. \"json\"). Defaults to text.
	UpOnly       bool   `json:"up_only"`
	Privileged   bool   `json:"privileged"`
}

func GetInterfaces(argsJSON []byte) (string, error) {
	var args GetInterfacesArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	cmdArgs := []string{"address", "show"}
	if args.UpOnly {
		cmdArgs = append(cmdArgs, "up")
	}

	cmd := exec.Command("ip", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ip error: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
