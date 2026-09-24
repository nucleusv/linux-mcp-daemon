package logging

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func capture(t *testing.T, c Config) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	SetOutput(&buf, c)
	t.Cleanup(func() { SetOutput(&bytes.Buffer{}, Config{}) })
	return &buf
}

func TestLevels(t *testing.T) {
	no := false
	cases := []struct {
		cfg  Config
		want []string // messages expected, of: debug info warn error audit access
	}{
		{Config{}, []string{"info", "warn", "error", "audit", "access"}},
		{Config{Level: "debug"}, []string{"debug", "info", "warn", "error", "audit", "access"}},
		{Config{Level: "warn"}, []string{"warn", "error", "audit", "access"}},
		{Config{Level: "error", AccessLog: &no}, []string{"error", "audit"}},
	}
	for _, c := range cases {
		buf := capture(t, c.cfg)
		Debug("debug")
		Info("info")
		Warn("warn")
		Error("error")
		Audit("audit")
		Access("uri", "/sse")
		out := buf.String()
		for _, m := range []string{"debug", "info", "warn", "error", "audit", "access"} {
			has := strings.Contains(out, " "+strings.ToUpper(levelOf(m))+" ") && strings.Contains(out, m)
			if has != contains(c.want, m) {
				t.Errorf("level %q: message %q logged=%t, want %t\n%s", c.cfg.Level, m, has, !has, out)
			}
		}
		if !strings.Contains(out, "audit=true") {
			t.Errorf("audit line lacks audit=true:\n%s", out)
		}
	}
}

func TestJSONFormat(t *testing.T) {
	buf := capture(t, Config{Format: "json"})
	Info("tool call", "user", "alice", "tool", "files/list")
	Audit("tool call", "user", "alice", "tool", "files/chmod")
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("not JSON: %q: %v", line, err)
		}
		if m["user"] != "alice" {
			t.Errorf("fields lost: %v", m)
		}
	}
}

func TestReconfigureAtRuntime(t *testing.T) {
	buf := capture(t, Config{Level: "warn"})
	Info("hidden")
	Configure(Config{Level: "debug"})
	Debug("shown")
	if strings.Contains(buf.String(), "hidden") || !strings.Contains(buf.String(), "shown") {
		t.Errorf("level change not applied:\n%s", buf.String())
	}
}

func TestValidate(t *testing.T) {
	for _, c := range []Config{{Level: "verbose"}, {Format: "xml"}} {
		if c.Validate() == nil {
			t.Errorf("%+v accepted", c)
		}
	}
	if err := (Config{Level: "DEBUG", Format: "json"}).Validate(); err != nil {
		t.Error(err)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Fatal must not be filtered out by the level: run it in a subprocess.
func TestFatalIgnoresLevel(t *testing.T) {
	if os.Getenv("LOGGING_FATAL_CHILD") == "1" {
		SetOutput(os.Stdout, Config{Level: "error"})
		Info("hidden")
		Fatal("cannot serve HTTP", "addr", ":9091")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatalIgnoresLevel")
	cmd.Env = append(os.Environ(), "LOGGING_FATAL_CHILD=1")
	out, err := cmd.Output()
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Fatalf("want exit 1, got %v", err)
	}
	if !strings.Contains(string(out), `ERROR cannot serve HTTP addr=:9091`) || strings.Contains(string(out), "hidden") {
		t.Errorf("fatal line missing or level not applied:\n%s", out)
	}
}

// levelOf is the level each test message is logged at.
func levelOf(m string) string {
	switch m {
	case "audit", "access":
		return "info"
	}
	return m
}

func TestTextFormat(t *testing.T) {
	buf := capture(t, Config{})
	Info("tool call", "user", "alice", "args", `{"path":"/tmp/a b"}`)
	Error("cannot serve HTTP", "err", "line1\nline2")
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines (values with newlines stay on one line), got:\n%s", buf.String())
	}
	re := regexp.MustCompile(`^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{3}(Z|[+-]\d\d:\d\d) INFO  tool call user=alice args="\{\\"path\\":\\"/tmp/a b\\"\}"$`)
	if !re.MatchString(lines[0]) {
		t.Errorf("info line: %q", lines[0])
	}
	if !strings.Contains(lines[1], " ERROR cannot serve HTTP err=") || strings.Contains(buf.String(), "time=") || strings.Contains(buf.String(), "level=") || strings.Contains(buf.String(), "msg=") {
		t.Errorf("unexpected format:\n%s", buf.String())
	}
}
