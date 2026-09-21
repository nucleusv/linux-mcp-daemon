package dmi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DMIData struct {
	SysVendor      string `json:"sys_vendor"`
	ProductName    string `json:"product_name"`
	ProductVersion string `json:"product_version"`
	BiosVersion    string `json:"bios_version"`
	BiosDate       string `json:"bios_date"`
	BoardName      string `json:"board_name"`
	BoardVendor    string `json:"board_vendor"`
}

func readSysfsValue(basePath, filename string) string {
	data, err := os.ReadFile(filepath.Join(basePath, filename))
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(data))
}

func ReadDMI(argsJSON []byte) (string, error) {
	dmiPath := "/sys/class/dmi/id"
	
	// If the directory doesn't exist, we aren't on a standard DMI platform (e.g. some ARM boards)
	if _, err := os.Stat(dmiPath); os.IsNotExist(err) {
		return "", fmt.Errorf("DMI data not available on this system")
	}

	data := DMIData{
		SysVendor:      readSysfsValue(dmiPath, "sys_vendor"),
		ProductName:    readSysfsValue(dmiPath, "product_name"),
		ProductVersion: readSysfsValue(dmiPath, "product_version"),
		BiosVersion:    readSysfsValue(dmiPath, "bios_version"),
		BiosDate:       readSysfsValue(dmiPath, "bios_date"),
		BoardName:      readSysfsValue(dmiPath, "board_name"),
		BoardVendor:    readSysfsValue(dmiPath, "board_vendor"),
	}

	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode DMI data: %v", err)
	}

	return string(output), nil
}
