package blocks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type GetBlocksArgs struct {
	All bool `json:"all"`
}

func GetBlocks(argsJSON []byte) (string, error) {
	var args GetBlocksArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	cmdArgs := []string{}
	if args.All {
		cmdArgs = append(cmdArgs, "-a")
	}

	cmd := exec.Command("lsblk", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("lsblk error: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
