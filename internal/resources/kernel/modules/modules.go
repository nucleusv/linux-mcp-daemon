package modules

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type KernelModule struct {
	Name        string   `json:"name"`
	Size        string   `json:"size"`
	UsedByCount string   `json:"used_by_count"`
	UsedBy      []string `json:"used_by"`
	State       string   `json:"state"`
	Address     string   `json:"address"`
}

func ReadModules(argsJSON []byte) (string, error) {
	data, err := os.ReadFile("/proc/modules")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/modules: %v", err)
	}

	var modules []KernelModule
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 6 {
			continue
		}

		usedByStr := parts[3]
		var usedBy []string
		if usedByStr != "-" {
			// UsedBy is comma separated, e.g. "nf_conntrack_netlink,"
			usedByStr = strings.TrimSuffix(usedByStr, ",")
			usedBy = strings.Split(usedByStr, ",")
		}

		modules = append(modules, KernelModule{
			Name:        parts[0],
			Size:        parts[1],
			UsedByCount: parts[2],
			UsedBy:      usedBy,
			State:       parts[4],
			Address:     parts[5],
		})
	}

	output, err := json.MarshalIndent(modules, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode modules data: %v", err)
	}

	return string(output), nil
}
