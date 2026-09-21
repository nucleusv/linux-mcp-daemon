package blocks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type GetBlocksArgs struct {
	OutputFormat string `json:\"output_format,omitempty\"` // OutputFormat specifies the desired output format (e.g. \"json\"). Defaults to text.
	All          bool   `json:"all"`
}

func GetBlocks(argsJSON []byte) (string, error) {
	var args GetBlocksArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	// Read native /sys/class/block instead of lsblk
	entries, err := os.ReadDir("/sys/class/block")
	if err != nil {
		return "", fmt.Errorf("failed to read /sys/class/block: %v", err)
	}

	var blocks []map[string]interface{}
	for _, entry := range entries {
		devName := entry.Name()
		if !args.All && strings.HasPrefix(devName, "loop") {
			continue // skip loop devices by default like lsblk
		}
		
		blockInfo := map[string]interface{}{
			"name": devName,
		}

		// Size in 512-byte sectors
		sizeStr, err := os.ReadFile(filepath.Join("/sys/class/block", devName, "size"))
		if err == nil {
			if sectors, err := strconv.ParseUint(strings.TrimSpace(string(sizeStr)), 10, 64); err == nil {
				blockInfo["size_bytes"] = sectors * 512
			}
		}

		// Read-only flag
		roStr, err := os.ReadFile(filepath.Join("/sys/class/block", devName, "ro"))
		if err == nil {
			blockInfo["ro"] = strings.TrimSpace(string(roStr)) == "1"
		}

		// Removable flag
		rmStr, err := os.ReadFile(filepath.Join("/sys/class/block", devName, "removable"))
		if err == nil {
			blockInfo["rm"] = strings.TrimSpace(string(rmStr)) == "1"
		}

		blocks = append(blocks, blockInfo)
	}

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		b, _ := json.Marshal(blocks)
		return string(b), nil
	}

	// Format text output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-12s %-12s %-6s %-6s\n", "NAME", "SIZE(BYTES)", "RO", "RM"))
	for _, b := range blocks {
		sb.WriteString(fmt.Sprintf("%-12s %-12v %-6v %-6v\n", b["name"], b["size_bytes"], b["ro"], b["rm"]))
	}

	return sb.String(), nil
}
