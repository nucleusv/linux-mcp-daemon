package deleteprocess

import (
	"fmt"
	"os"
	"testing"
)

func TestProtectedPIDsAndSignals(t *testing.T) {
	for _, pid := range []int{1, os.Getppid(), os.Getpid()} {
		if _, err := Delete([]byte(fmt.Sprintf(`{"pid":%d}`, pid))); err == nil {
			t.Errorf("signalled protected pid %d", pid)
		}
	}
	for _, sig := range []string{"-1", "9; rm -rf /", "SIGBOGUS", "--"} {
		if _, err := Delete([]byte(fmt.Sprintf(`{"pid":999999,"signal":%q}`, sig))); err == nil || !contains(err.Error(), "invalid signal") {
			t.Errorf("signal %q not rejected as invalid: %v", sig, err)
		}
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
