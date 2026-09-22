package interfaces

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Read returns network interfaces. If targetName is provided, it returns only that interface.
func Read(targetName string) (string, string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", "", fmt.Errorf("net.Interfaces failed: %v", err)
	}

	var resultList []map[string]interface{}
	for _, iface := range ifaces {
		if targetName != "" && iface.Name != targetName {
			continue
		}
		addrs, _ := iface.Addrs()
		var addrList []string
		for _, addr := range addrs {
			addrList = append(addrList, addr.String())
		}
		resultList = append(resultList, map[string]interface{}{
			"index":      iface.Index,
			"name":       iface.Name,
			"mac":        iface.HardwareAddr.String(),
			"mtu":        iface.MTU,
			"flags":      iface.Flags.String(),
			"addresses":  addrList,
			"statistics": getInterfaceStats(iface.Name),
		})
	}

	if targetName != "" {
		if len(resultList) == 0 {
			return "", "", fmt.Errorf("interface %s not found", targetName)
		}
		b, _ := json.MarshalIndent(resultList[0], "", "  ")
		return string(b), "application/json", nil
	}

	b, _ := json.MarshalIndent(resultList, "", "  ")
	return string(b), "application/json", nil
}

func readUint64FromFile(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return val
}

func getInterfaceStats(ifaceName string) map[string]uint64 {
	basePath := filepath.Join("/sys/class/net", ifaceName, "statistics")
	
	// If the sysfs directory doesn't exist (e.g. on macOS or no sysfs), return nil
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return nil
	}

	stats := make(map[string]uint64)
	metrics := []string{
		"rx_bytes", "rx_packets", "rx_errors", "rx_dropped",
		"tx_bytes", "tx_packets", "tx_errors", "tx_dropped",
	}

	for _, metric := range metrics {
		stats[metric] = readUint64FromFile(filepath.Join(basePath, metric))
	}

	return stats
}
