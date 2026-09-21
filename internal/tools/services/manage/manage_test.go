package manage

import (
	"encoding/json"
	"testing"
)

func TestManage(t *testing.T) {
	// DBus tests usually fail in CI without systemd running, so we just test argument validation
	args := ManageServiceArgs{Action: "start"}
	argsJSON, _ := json.Marshal(args)
	_, err := Manage(argsJSON)
	if err == nil || err.Error() != "service argument is required" {
		t.Errorf("Expected 'service argument is required', got: %v", err)
	}
}
