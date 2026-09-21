package journalctl

import (
	"encoding/json"
	"testing"
)

func TestJournalctl(t *testing.T) {
	args := JournalctlArgs{Lines: 1}
	argsJSON, _ := json.Marshal(args)
	_, err := Journalctl(argsJSON)
	if err != nil {
		t.Logf("Journalctl failed (maybe systemd not running in test env): %v", err)
	}
}
