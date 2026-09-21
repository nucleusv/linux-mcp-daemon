package arp

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ARPArgs defines the parameters for the network/arp tool.
type ARPArgs struct {
	Interface    string `json:"interface,omitempty"`     // Interface filters the ARP cache by a specific network interface (e.g. "eth0").
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
}

type ARPEntry struct {
	IPAddress string `json:"ip_address"`
	HWType    string `json:"hw_type"`
	Flags     string `json:"flags"`
	HWAddress string `json:"hw_address"`
	Mask      string `json:"mask"`
	Device    string `json:"device"`
}

func ARP(argsJSON []byte) (string, error) {
	var args ARPArgs
	if len(argsJSON) > 0 && string(argsJSON) != "{}" {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	data, err := os.ReadFile("/proc/net/arp")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/net/arp: %v", err)
	}

	var entries []ARPEntry
	lines := strings.Split(string(data), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // Skip header and empty lines
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		entry := ARPEntry{
			IPAddress: fields[0],
			HWType:    fields[1],
			Flags:     fields[2],
			HWAddress: fields[3],
			Mask:      fields[4],
			Device:    fields[5],
		}

		if args.Interface != "" && entry.Device != args.Interface {
			continue
		}

		entries = append(entries, entry)
	}

	out, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode response: %v", err)
	}

	return string(out), nil
}
