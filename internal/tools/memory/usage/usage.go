package usage

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type GetUsageArgs struct {
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
	Detailed     bool   `json:"detailed"`
}

func Usage(argsJSON []byte) (string, error) {
	var args GetUsageArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	content, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/meminfo: %v", err)
	}

	if args.Detailed && (args.OutputFormat == "" || args.OutputFormat == "text") {
		return string(content), nil
	}

	// Parse basic memory info
	mem := make(map[string]uint64)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			key := strings.TrimSuffix(parts[0], ":")
			val, _ := strconv.ParseUint(parts[1], 10, 64)
			mem[key] = val * 1024 // convert from kB to bytes
		}
	}

	// Calculate standard "free" metrics
	total := mem["MemTotal"]
	free := mem["MemFree"]
	available := mem["MemAvailable"]
	buffers := mem["Buffers"]
	cached := mem["Cached"]
	sReclaimable := mem["SReclaimable"]

	buffCache := buffers + cached + sReclaimable
	used := total - free - buffCache

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		data := map[string]interface{}{
			"total":     total,
			"used":      used,
			"free":      free,
			"shared":    mem["Shmem"],
			"buffCache": buffCache,
			"available": available,
		}
		b, _ := json.Marshal(data)
		return string(b), nil
	}

	// Format text output similar to `free`
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-12s %-12s %-12s %-12s %-12s %-12s %-12s\n", "TYPE", "TOTAL", "USED", "FREE", "SHARED", "BUFF/CACHE", "AVAILABLE"))
	sb.WriteString(fmt.Sprintf("%-12s %-12d %-12d %-12d %-12d %-12d %-12d\n", "Mem:", total, used, free, mem["Shmem"], buffCache, available))

	return sb.String(), nil
}
