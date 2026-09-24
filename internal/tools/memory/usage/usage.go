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

	swapTotal, swapFree := mem["SwapTotal"], mem["SwapFree"]
	swapUsed := swapTotal - swapFree

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		data := map[string]interface{}{
			"total":      total,
			"used":       used,
			"free":       free,
			"shared":     mem["Shmem"],
			"buffCache":  buffCache,
			"available":  available,
			"swap_total": swapTotal,
			"swap_used":  swapUsed,
			"swap_free":  swapFree,
		}
		b, _ := json.Marshal(data)
		return string(b), nil
	}

	// Text like `free -h`: sizes in Ki/Mi/Gi, a Swap row.
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-6s %10s %10s %10s %10s %10s %10s\n", "", "total", "used", "free", "shared", "buff/cache", "available"))
	sb.WriteString(fmt.Sprintf("%-6s %10s %10s %10s %10s %10s %10s\n", "Mem:", human(total), human(used), human(free), human(mem["Shmem"]), human(buffCache), human(available)))
	sb.WriteString(fmt.Sprintf("%-6s %10s %10s %10s\n", "Swap:", human(swapTotal), human(swapUsed), human(swapFree)))
	return sb.String(), nil
}

// human formats bytes the way `free -h` does: 1.8Gi, 512Mi, 0B.
func human(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	v, exp := float64(b)/unit, 0
	for v >= unit && exp < 4 {
		v /= unit
		exp++
	}
	suffix := []string{"Ki", "Mi", "Gi", "Ti", "Pi"}[exp]
	if v < 10 {
		return fmt.Sprintf("%.1f%s", v, suffix)
	}
	return fmt.Sprintf("%.0f%s", v, suffix)
}
