package docker

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func frame(typ byte, payload string) string {
	h := make([]byte, 8)
	h[0] = typ
	binary.BigEndian.PutUint32(h[4:], uint32(len(payload)))
	return string(h) + payload
}

func TestDemux(t *testing.T) {
	in := frame(1, "out one\n") + frame(2, "err one\n") + frame(1, "out two\n")
	stdout, stderr, err := Demux(strings.NewReader(in), 1<<20)
	if err != nil {
		t.Fatalf("Demux: %v", err)
	}
	if stdout != "out one\nout two\n" {
		t.Errorf("stdout = %q", stdout)
	}
	if stderr != "err one\n" {
		t.Errorf("stderr = %q", stderr)
	}
}

// A TTY container's stream carries no frame headers at all; it must come back
// as stdout rather than being mangled by the framed parser.
func TestDemuxUnframedTTY(t *testing.T) {
	stdout, stderr, err := Demux(strings.NewReader("plain tty output\n"), 1<<20)
	if err != nil {
		t.Fatalf("Demux: %v", err)
	}
	if stdout != "plain tty output\n" || stderr != "" {
		t.Errorf("got (%q, %q)", stdout, stderr)
	}
}

// Hitting the byte limit mid-frame must truncate, not panic or discard.
func TestDemuxTruncatedFrame(t *testing.T) {
	in := frame(1, "hello world")
	stdout, _, err := Demux(strings.NewReader(in), int64(len(in)-6))
	if err != nil {
		t.Fatalf("Demux: %v", err)
	}
	if stdout != "hello" {
		t.Errorf("stdout = %q, want the first 5 bytes", stdout)
	}
}

func TestSocketMissingError(t *testing.T) {
	c := New("/nonexistent/docker.sock", time.Second)
	err := c.GetJSON("/containers/json", &struct{}{})
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"/nonexistent/docker.sock", "is Docker installed", "worker.docker_socket"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}

func TestQ(t *testing.T) {
	if got := Q(); got != "" {
		t.Errorf("Q() = %q, want empty", got)
	}
	if got := Q("all", "true", "limit", ""); got != "?all=true" {
		t.Errorf("Q dropped the wrong pair: %q", got)
	}
}

// A containerized privileged worker must dial through /proc/1/root: only its
// main thread joined the host mount namespace, and net/http dials on another.
func TestDialPathHostRoot(t *testing.T) {
	if got := dialPath("/var/run/docker.sock"); got != "/var/run/docker.sock" {
		t.Errorf("without MCPD_HOST_ROOT the path must be untouched, got %q", got)
	}
	t.Setenv("MCPD_HOST_ROOT", "1")

	// A symlinked directory stands in for the host's /var/run -> /run.
	dir := t.TempDir()
	real, err := filepath.EvalSymlinks(dir) // macOS /tmp is itself a symlink
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(real+"/run", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(real+"/run/docker.sock", nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real+"/run", real+"/var-run"); err != nil {
		t.Fatal(err)
	}
	if got, want := dialPath(real+"/var-run/docker.sock"), "/proc/1/root"+real+"/run/docker.sock"; got != want {
		t.Errorf("dialPath = %q, want %q", got, want)
	}
	if got := dialPath("/proc/1/root/run/docker.sock"); got != "/proc/1/root/run/docker.sock" {
		t.Errorf("an already-prefixed path must not be prefixed twice: %q", got)
	}
}
