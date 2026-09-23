package usage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExcluded(t *testing.T) {
	cases := []struct {
		patterns   []string
		path, name string
		want       bool
	}{
		{[]string{"/proc"}, "/proc", "proc", true},
		{[]string{"/proc"}, "/proc/1/status", "status", true},
		{[]string{"/proc"}, "/processes", "processes", false},
		{[]string{"*.log"}, "/var/log/syslog.log", "syslog.log", true},
		{[]string{"/var/lib/*"}, "/var/lib/docker", "docker", true},
		{[]string{"node_modules"}, "/a/node_modules", "node_modules", true},
		{nil, "/etc", "etc", false},
	}
	for _, c := range cases {
		if got := excluded(c.patterns, c.path, c.name); got != c.want {
			t.Errorf("excluded(%v, %q) = %v, want %v", c.patterns, c.path, got, c.want)
		}
	}
}

func usage(t *testing.T, args map[string]interface{}) map[string]interface{} {
	t.Helper()
	args["output_format"] = "json"
	args["apparent_size"] = true // deterministic across filesystems
	b, _ := json.Marshal(args)
	out, err := Usage(b)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	json.Unmarshal([]byte(out), &m)
	return m
}

func TestThresholdDoesNotShrinkTotals(t *testing.T) {
	// Regression: threshold used to be applied per file while summing,
	// so small files vanished from every directory total.
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "big"), 0o755)
	os.MkdirAll(filepath.Join(root, "small"), 0o755)
	// Sizes are chosen to stay clear of a directory entry's own apparent
	// size, which is 4096 on Linux (ext4/overlayfs) but tiny on macOS APFS.
	os.WriteFile(filepath.Join(root, "big", "a"), make([]byte, 30000), 0o644)
	for i := 0; i < 10; i++ {
		os.WriteFile(filepath.Join(root, "big", "s"+string(rune('0'+i))), make([]byte, 100), 0o644)
	}
	os.WriteFile(filepath.Join(root, "small", "b"), make([]byte, 100), 0o644)

	m := usage(t, map[string]interface{}{"path": root, "max_depth": 1, "threshold": 20000})
	sizes := m["directory_sizes"].(map[string]interface{})
	big := sizes[filepath.Join(root, "big")].(float64)
	if big < 31000 { // 30000 + 10*100 (+ the dir entry itself)
		t.Errorf("big/ = %v, want >= 31000 (small files must still count)", big)
	}
	if _, shown := sizes[filepath.Join(root, "small")]; shown {
		t.Errorf("small/ should be hidden by threshold, got %v", sizes)
	}
}

func TestExcludeFullPathAndSorted(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"a", "b", "skip"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
	}
	os.WriteFile(filepath.Join(root, "a", "f"), make([]byte, 100), 0o644)
	os.WriteFile(filepath.Join(root, "b", "f"), make([]byte, 5000), 0o644)
	os.WriteFile(filepath.Join(root, "skip", "f"), make([]byte, 9000), 0o644)

	b, _ := json.Marshal(map[string]interface{}{"path": root, "max_depth": 1, "apparent_size": true,
		"exclude": []string{filepath.Join(root, "skip")}})
	out, err := Usage(b)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "skip") {
		t.Errorf("excluded dir still listed:\n%s", out)
	}
	ia, ib := strings.Index(out, filepath.Join(root, "a")+"\n"), strings.Index(out, filepath.Join(root, "b")+"\n")
	if ib < 0 || ia < 0 || ib > ia {
		t.Errorf("want b (larger) listed before a:\n%s", out)
	}
}

func TestHardlinksCountedOnce(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "f")
	os.WriteFile(f, make([]byte, 5000), 0o644)
	if err := os.Link(f, filepath.Join(root, "g")); err != nil {
		t.Skip("hard links unsupported:", err)
	}
	m := usage(t, map[string]interface{}{"path": root})
	if total := m["total_size"].(float64); total >= 10000 {
		t.Errorf("total %v counts the hard link twice", total)
	}
}
