package rpc

import (
	"strings"
	"testing"
)

func TestRedactArgsNested(t *testing.T) {
	out := redactArgs([]byte(`{"url":"https://x","headers":{"Authorization":"Bearer SECRET1"},"body":"SECRET2","nested":{"deep":{"password":"SECRET3"}},"list":[{"token":"SECRET4"}]}`))
	for _, secret := range []string{"SECRET1", "SECRET2", "SECRET3", "SECRET4"} {
		if strings.Contains(out, secret) {
			t.Errorf("%s leaked into log line: %s", secret, out)
		}
	}
	if !strings.Contains(out, "https://x") {
		t.Errorf("non-sensitive field lost: %s", out)
	}
}
