package hostname

import (
	"fmt"
	"os"
)

// Read returns the system hostname
func Read() (string, string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", "", fmt.Errorf("os.Hostname failed: %v", err)
	}
	return hostname, "text/plain", nil
}
