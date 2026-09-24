package worker

import "testing"

func TestLooksLikePermissionError(t *testing.T) {
	for msg, want := range map[string]bool{
		"failed to open file: open /etc/shadow: permission denied":                    true,
		"/tmp/x: operation not permitted":                                             true,
		"No journal files were opened due to insufficient permissions.":               true,
		"not permitted to signal process 66594 (owned by another user)":               true,
		"failed to open file: open /tmp/x: no such file or directory":                 false,
		`invalid mode "999": expected octal (0755) or symbolic ([ugoa][+-=][rwxXst])`: false,
	} {
		if got := looksLikePermissionError(msg); got != want {
			t.Errorf("%q: got %t", msg, got)
		}
	}
}
