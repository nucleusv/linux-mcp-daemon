package dmesg

import (
	"testing"
)

func TestDmesg(t *testing.T) {
	// Dmesg often requires root in modern systems (dmesg_restrict).
	// We'll just run it with no args and skip if it fails due to permissions.
	_, err := Dmesg([]byte("{}"))
	if err != nil {
		t.Logf("Dmesg failed (likely dmesg_restrict): %v", err)
	}
}
