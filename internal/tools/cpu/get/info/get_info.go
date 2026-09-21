package info

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type GetInfoArgs struct {
	TopologyOnly bool `json:"topology_only"`
}

func GetInfo(argsJSON []byte) (string, error) {
	var args GetInfoArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	cmdArgs := []string{}
	if args.TopologyOnly {
		// Output parsable basic topologies
		cmdArgs = append(cmdArgs, "-e", "CPU,CORE,SOCKET,NODE,ONLINE")
	}

	cmd := exec.Command("lscpu", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("lscpu error: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
