package read

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProcessReadArgs defines the parameters for the process://{pid}/* resource.
type ProcessReadArgs struct {
	PID    int    `json:"pid"`
	Target string `json:"target"` // "status", "cmdline", "environ"
}

func Read(argsJSON []byte) (string, error) {
	var args ProcessReadArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if args.PID <= 0 {
		return "", fmt.Errorf("invalid pid: %d", args.PID)
	}

	switch args.Target {
	case "status":
		content, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", args.PID))
		if err != nil {
			return "", fmt.Errorf("failed to read status: %v", err)
		}
		return string(content), nil

	case "cmdline":
		content, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", args.PID))
		if err != nil {
			return "", fmt.Errorf("failed to read cmdline: %v", err)
		}
		// cmdline is null-byte separated
		argsList := strings.Split(string(content), "\x00")
		// the last element might be empty due to trailing null byte
		if len(argsList) > 0 && argsList[len(argsList)-1] == "" {
			argsList = argsList[:len(argsList)-1]
		}
		out, _ := json.MarshalIndent(argsList, "", "  ")
		return string(out), nil

	case "environ":
		content, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", args.PID))
		if err != nil {
			return "", fmt.Errorf("failed to read environ: %v", err)
		}
		// environ is null-byte separated KEY=VALUE pairs
		envList := strings.Split(string(content), "\x00")
		envMap := make(map[string]string)
		for _, env := range envList {
			if env == "" {
				continue
			}
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}
		out, _ := json.MarshalIndent(envMap, "", "  ")
		return string(out), nil

	default:
		return "", fmt.Errorf("unsupported target: %s", args.Target)
	}
}

// CleanUp cleans up string paths
func CleanUp(s string) string {
	return filepath.Clean(s)
}
