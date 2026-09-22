package interfaces

import (
	"encoding/json"
	"os"
	"testing"
)

func TestRead(t *testing.T) {
	// Read all interfaces
	res, mime, err := Read("")
	if err != nil {
		t.Fatalf("Read all failed: %v", err)
	}
	if mime != "application/json" {
		t.Errorf("Expected application/json, got %s", mime)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(res), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if len(parsed) == 0 {
		t.Fatalf("No interfaces returned")
	}

	// Verify loopback 'lo' (or 'lo0' on macOS)
	foundLo := false
	for _, iface := range parsed {
		name := iface["name"].(string)
		if name == "lo" || name == "lo0" {
			foundLo = true
			// Check if statistics are present when running on Linux
			if _, err := os.Stat("/sys/class/net"); !os.IsNotExist(err) {
				if iface["statistics"] == nil {
					t.Errorf("Expected statistics object, got nil")
				} else {
					stats := iface["statistics"].(map[string]interface{})
					if _, ok := stats["rx_bytes"]; !ok {
						t.Errorf("Missing rx_bytes in statistics")
					}
				}
			}
			break
		}
	}
	if !foundLo {
		t.Errorf("Could not find loopback interface")
	}
}
