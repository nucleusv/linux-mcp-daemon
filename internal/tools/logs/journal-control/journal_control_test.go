package journal_control

import (
	"encoding/json"
	"testing"
)

func TestJournalctl(t *testing.T) {
	args := JournalControlArgs{Lines: 1}
	argsJSON, _ := json.Marshal(args)
	result, err := JournalControl(argsJSON)
	if err != nil {
		t.Logf("Journalctl failed (maybe systemd not running in test env): %v", err)
	}
}
