//go:build linux

package chmod

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func modeOf(t *testing.T, p string) uint32 {
	t.Helper()
	var st syscall.Stat_t
	if err := syscall.Lstat(p, &st); err != nil {
		t.Fatal(err)
	}
	return st.Mode & 0o7777
}

func run(t *testing.T, args map[string]interface{}) (string, error) {
	t.Helper()
	b, _ := json.Marshal(args)
	return Chmod(b)
}

// TestParseModeMatchesGNUChmod applies each spec with the real chmod
// binary and with ParseMode, on both a file and a directory.
func TestParseModeMatchesGNUChmod(t *testing.T) {
	if out, err := exec.Command("chmod", "--version").Output(); err != nil || !strings.Contains(string(out), "GNU") {
		t.Skip("GNU chmod not available")
	}
	specs := []string{"644", "755", "4755", "2750", "1777", "00755", "06755", "u+x", "go-w", "a=r", "u+x,g-r", "+X", "a+X", "u+s", "g+s", "+t", "u=rx", "u=rwx", "o=", "ug=rw,o-rwx", "a-x", "u-s"}
	starts := []uint32{0o644, 0o755, 0o4755, 0o2775, 0o600, 0o1777}
	dir := t.TempDir()
	for _, isDir := range []bool{false, true} {
		for _, start := range starts {
			for _, spec := range specs {
				p := filepath.Join(dir, "x")
				os.RemoveAll(p)
				if isDir {
					os.Mkdir(p, 0o700)
				} else {
					os.WriteFile(p, nil, 0o600)
				}
				if err := os.Chmod(p, 0); err != nil {
					t.Fatal(err)
				}
				exec.Command("chmod", strconv.FormatUint(uint64(start), 8), p).Run()
				before := modeOf(t, p)
				if err := exec.Command("chmod", spec, p).Run(); err != nil {
					t.Fatalf("chmod %s: %v", spec, err)
				}
				want := modeOf(t, p)
				f, err := ParseMode(spec)
				if err != nil {
					t.Fatalf("ParseMode(%q): %v", spec, err)
				}
				if got := f(before, isDir); got != want {
					t.Errorf("dir=%v start=%04o spec=%q: ours %04o, GNU chmod %04o", isDir, before, spec, got, want)
				}
			}
		}
	}
}

func TestChmodNeverFollowsSymlinks(t *testing.T) {
	base := t.TempDir()
	outside := filepath.Join(base, "outside")
	os.WriteFile(outside, nil, 0o600)
	os.Mkdir(filepath.Join(base, "realdir"), 0o755)
	inside := filepath.Join(base, "tree")
	os.Mkdir(inside, 0o755)
	os.WriteFile(filepath.Join(inside, "file"), nil, 0o600)
	os.Symlink(outside, filepath.Join(inside, "link-to-outside"))
	os.Symlink(filepath.Join(base, "realdir"), filepath.Join(base, "dirlink"))

	// The target itself is a symlink: refused.
	if _, err := run(t, map[string]interface{}{"path": filepath.Join(inside, "link-to-outside"), "mode": "0777"}); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Errorf("symlink target not refused: %v", err)
	}
	// A directory component is a symlink: refused.
	if _, err := run(t, map[string]interface{}{"path": filepath.Join(base, "dirlink", "anything"), "mode": "0777"}); err == nil {
		t.Error("symlinked directory component not refused")
	}
	// Recursive: the in-tree file changes, the symlinked outside file doesn't.
	out, err := run(t, map[string]interface{}{"path": inside, "mode": "0750", "recursive": true})
	if err != nil {
		t.Fatal(err)
	}
	if modeOf(t, filepath.Join(inside, "file")) != 0o750 {
		t.Errorf("in-tree file not changed:\n%s", out)
	}
	if modeOf(t, outside) != 0o600 {
		t.Errorf("recursive chmod followed a symlink out of the tree - outside file is now %04o", modeOf(t, outside))
	}
	if !strings.Contains(out, "skipped 1 symlink") {
		t.Errorf("skipped symlink not reported:\n%s", out)
	}
	// Plain case, and the "unchanged" message.
	p := filepath.Join(inside, "file")
	if out, err := run(t, map[string]interface{}{"path": p, "mode": "u+w,g-x"}); err != nil || !strings.Contains(out, "0750 (-rwxr-x---) -> 0740 (-rwxr-----)") {
		t.Errorf("chmod: %q %v", out, err)
	}
	if out, _ := run(t, map[string]interface{}{"path": p, "mode": "740"}); !strings.Contains(out, "unchanged") {
		t.Errorf("no-op not reported as unchanged: %q", out)
	}
	if _, err := run(t, map[string]interface{}{"path": "relative/path", "mode": "644"}); err == nil {
		t.Error("relative path accepted")
	}
}
