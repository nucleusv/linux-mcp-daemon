package pci

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PCIDevice struct {
	Slot     string `json:"slot"`
	VendorID string `json:"vendor_id"`
	DeviceID string `json:"device_id"`
	Class    string `json:"class"`
}

func ReadPCI(argsJSON []byte) (string, error) {
	sysfsPath := "/sys/bus/pci/devices"
	entries, err := os.ReadDir(sysfsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %v", sysfsPath, err)
	}

	var devices []PCIDevice

	for _, entry := range entries {
		devPath := filepath.Join(sysfsPath, entry.Name())

		vid, _ := os.ReadFile(filepath.Join(devPath, "vendor"))
		did, _ := os.ReadFile(filepath.Join(devPath, "device"))
		cls, _ := os.ReadFile(filepath.Join(devPath, "class"))

		vendor := strings.TrimSpace(string(vid))
		device := strings.TrimSpace(string(did))

		if vendor == "" || device == "" {
			continue
		}

		devices = append(devices, PCIDevice{
			Slot:     entry.Name(),
			VendorID: vendor,
			DeviceID: device,
			Class:    strings.TrimSpace(string(cls)),
		})
	}

	output, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode PCI data: %v", err)
	}

	return string(output), nil
}
