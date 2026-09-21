package load_average

import (
	"encoding/json"
	"fmt"
	"syscall"
)

type GetLoadAverageArgs struct {
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
}

func LoadAverage(argsJSON []byte) (string, error) {
	var args GetLoadAverageArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("failed to parse args: %v", err)
	}
	// Use the native Linux syscall for system info
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return "", fmt.Errorf("failed to get sysinfo: %v", err)
	}

	// Loads are stored as fixed-point values shifted by 16 bits
	const shift = 65536.0
	load1 := float64(info.Loads[0]) / shift
	load5 := float64(info.Loads[1]) / shift
	load15 := float64(info.Loads[2]) / shift

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		data := map[string]interface{}{
			"1_min":  load1,
			"5_min":  load5,
			"15_min": load15,
		}
		jsonB, _ := json.Marshal(data)
		return string(jsonB), nil
	}

	return fmt.Sprintf("Load Average: %.2f, %.2f, %.2f", load1, load5, load15), nil
}
