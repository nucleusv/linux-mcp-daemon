package list

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type GetInfoArgs struct {
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
	TopologyOnly bool   `json:"topology_only"`
}

func List(argsJSON []byte) (string, error) {
	var args GetInfoArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	// Read native /proc/cpuinfo instead of lscpu
	content, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/cpuinfo: %v", err)
	}

	// Parse basic CPU info
	var processors []map[string]string
	var currentProc map[string]string
	
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if currentProc != nil {
				processors = append(processors, currentProc)
				currentProc = nil
			}
			continue
		}
		
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			if currentProc == nil {
				currentProc = make(map[string]string)
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			currentProc[key] = val
		}
	}
	if currentProc != nil {
		processors = append(processors, currentProc)
	}

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		b, _ := json.Marshal(processors)
		return string(b), nil
	}

	// Format text output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CPU Information (Total Processors: %d)\n", len(processors)))
	if len(processors) > 0 {
		first := processors[0]

		vendor := first["vendor_id"]
		if vendor == "" {
			vendor = first["CPU implementer"]
		}
		model := first["model name"]
		if model == "" {
			model = first["CPU architecture"]
		}
		mhz := first["cpu MHz"]
		if mhz == "" {
			mhz = first["BogoMIPS"]
		}
		cache := first["cache size"]
		if cache == "" {
			cache = "N/A"
		}

		sb.WriteString(fmt.Sprintf("Vendor ID: %s\n", vendor))
		sb.WriteString(fmt.Sprintf("Model Name: %s\n", model))
		sb.WriteString(fmt.Sprintf("CPU MHz/BogoMIPS: %s\n", mhz))
		sb.WriteString(fmt.Sprintf("Cache Size: %s\n", cache))
	}
	
	return sb.String(), nil
}
