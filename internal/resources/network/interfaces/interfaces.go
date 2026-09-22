package interfaces

import (
	"encoding/json"
	"fmt"
	"net"
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
			"index":     iface.Index,
			"name":      iface.Name,
			"mac":       iface.HardwareAddr.String(),
			"mtu":       iface.MTU,
			"flags":     iface.Flags.String(),
			"addresses": addrList,
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
