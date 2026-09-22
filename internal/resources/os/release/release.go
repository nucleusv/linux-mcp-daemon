package release

import (
	"fmt"
	"os"
)

// Read returns the content of /etc/os-release
func Read() (string, string, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", "", fmt.Errorf("failed to read /etc/os-release: %v", err)
	}
	return string(data), "text/plain", nil
}
