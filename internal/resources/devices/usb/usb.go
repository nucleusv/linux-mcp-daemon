package usb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type USBDevice struct {
	BusID        string `json:"bus_id"`
	VendorID     string `json:"vendor_id"`
	ProductID    string `json:"product_id"`
	Manufacturer string `json:"manufacturer"`
	Product      string `json:"product"`
}

func ReadUSB(argsJSON []byte) (string, error) {
	sysfsPath := "/sys/bus/usb/devices"
	entries, err := os.ReadDir(sysfsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %v", sysfsPath, err)
	}

	var devices []USBDevice

	for _, entry := range entries {
		// Skip interfaces (they contain ':') and usbX hubs (they start with 'usb')
		if strings.Contains(entry.Name(), ":") {
			continue
		}

		devPath := filepath.Join(sysfsPath, entry.Name())

		vid, _ := os.ReadFile(filepath.Join(devPath, "idVendor"))
		pid, _ := os.ReadFile(filepath.Join(devPath, "idProduct"))
		mfg, _ := os.ReadFile(filepath.Join(devPath, "manufacturer"))
		prod, _ := os.ReadFile(filepath.Join(devPath, "product"))

		vendor := strings.TrimSpace(string(vid))
		productID := strings.TrimSpace(string(pid))

		if vendor == "" || productID == "" {
			continue
		}

		devices = append(devices, USBDevice{
			BusID:        entry.Name(),
			VendorID:     vendor,
			ProductID:    productID,
			Manufacturer: strings.TrimSpace(string(mfg)),
			Product:      strings.TrimSpace(string(prod)),
		})
	}

	output, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode USB data: %v", err)
	}

	return string(output), nil
}
