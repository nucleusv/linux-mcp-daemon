package deleteprocess

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
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

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestSignalsRealProcess(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skip("cannot start sleep:", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	out, err := Delete([]byte(fmt.Sprintf(`{"pid":%d,"signal":"term"}`, cmd.Process.Pid)))
	if err != nil || !contains(out, "SIGTERM") {
		t.Fatalf("Delete = %q, %v", out, err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("process still running after SIGTERM")
	}
	// Now it's gone: a clear "no such process", not a raw errno.
	if _, err := Delete([]byte(fmt.Sprintf(`{"pid":%d}`, cmd.Process.Pid))); err == nil || !contains(err.Error(), "no such process") {
		t.Errorf("signalling an exited PID: %v", err)
	}
}
