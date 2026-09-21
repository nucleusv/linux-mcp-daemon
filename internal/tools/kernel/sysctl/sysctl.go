package sysctl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// SysctlArgs defines the parameters for the kernel/sysctl tool.
type SysctlArgs struct {
	Key     string `json:"key"`               // Key is the kernel parameter to read or write (e.g. "net.ipv4.ip_forward"). Required unless ReadAll is true.
	Value   string `json:"value,omitempty"`   // Value is the value to write to the parameter. If provided, writes the value.
	ReadAll bool   `json:"read_all,omitempty"`// ReadAll reads all kernel parameters (sysctl -a).
}

func Sysctl(argsJSON []byte) (string, error) {
	var args SysctlArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	cmdArgs := []string{}

	if args.ReadAll {
		cmdArgs = append(cmdArgs, "-a")
	} else {
		if args.Key == "" {
			return "", fmt.Errorf("key argument is required unless read_all is true")
		}

		if args.Value != "" {
			// Write mode
			cmdArgs = append(cmdArgs, "-w", fmt.Sprintf("%s=%s", args.Key, args.Value))
		} else {
			// Read mode
			cmdArgs = append(cmdArgs, args.Key)
		}
	}

	cmd := exec.Command("sysctl", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("sysctl error: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
