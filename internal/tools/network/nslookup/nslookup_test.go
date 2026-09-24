package nslookup

import (
	"strings"
	"testing"
)

func TestInvalidInput(t *testing.T) {
	if _, err := Nslookup([]byte(`{"host": "example.com", "record_type": "BOGUS"}`)); err == nil || !strings.Contains(err.Error(), "unsupported record type") {
		t.Errorf("bogus type: %v", err)
	}
	if _, err := Nslookup([]byte(`{"host": " "}`)); err == nil {
		t.Error("empty host accepted")
	}
}

// .invalid never resolves (RFC 2606), so this needs no network to fail -
// but a resolver must be reachable to say so; skip when there isn't one.
func TestNoSuchHost(t *testing.T) {
	_, err := Nslookup([]byte(`{"host": "no-such-host.invalid", "record_type": "A"}`))
	if err == nil {
		t.Fatal("nonexistent host returned no error")
	}
	if !strings.Contains(err.Error(), "no such host") {
		t.Skipf("no resolver here: %v", err)
	}
}
