package status

import (
	"encoding/json"
	"testing"
)

func TestStatus(t *testing.T) {
	args := ServiceStatusArgs{}
	argsJSON, _ := json.Marshal(args)
	_, err := Status(argsJSON)
	if err == nil || err.Error() != "service argument is required" {
		t.Errorf("Expected 'service argument is required', got: %v", err)
	}
}
