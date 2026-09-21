package usage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type GetUsageArgs struct {
	Detailed bool `json:"detailed"`
}

func GetUsage(argsJSON []byte) (string, error) {
	var args GetUsageArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	if args.Detailed {
		content, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return "", fmt.Errorf("failed to read /proc/meminfo: %v", err)
		}
		return string(content), nil
	}

	cmd := exec.Command("free", "-h")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("free error: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
