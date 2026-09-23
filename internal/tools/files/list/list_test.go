package listfiles

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestModeString(t *testing.T) {
	cases := map[fs.FileMode]string{
		0o644:                                     "-rw-r--r--",
		fs.ModeDir | 0o755:                        "drwxr-xr-x",
		fs.ModeDir | fs.ModeSticky | 0o777:        "drwxrwxrwt",
		fs.ModeDir | fs.ModeSticky | 0o770:        "drwxrwx--T",
		fs.ModeSetuid | 0o755:                     "-rwsr-xr-x",
		fs.ModeSetuid | 0o644:                     "-rwSr--r--",
		fs.ModeSetgid | 0o2750:                    "-rwxr-s---",
		fs.ModeSymlink | 0o777:                    "lrwxrwxrwx",
		fs.ModeDevice | fs.ModeCharDevice | 0o666: "crw-rw-rw-",
		fs.ModeDevice | 0o660:                     "brw-rw----",
		fs.ModeNamedPipe | 0o644:                  "prw-r--r--",
		fs.ModeSocket | 0o755:                     "srwxr-xr-x",
	}
	for m, want := range cases {
		if got := modeString(m); got != want {
			t.Errorf("modeString(%v) = %q, want %q", m, got, want)
		}
	}
	if got := unixPerm(fs.ModeSetuid | fs.ModeSticky | 0o755); got != 0o5755 {
		t.Errorf("unixPerm = %o", got)
	}
}

func TestHumanSize(t *testing.T) {
	for n, want := range map[int64]string{0: "0", 1023: "1023", 1024: "1.0K", 4096: "4.0K", 4097: "4.1K", 12 * 1024: "12K", 1536 * 1024: "1.5M", 5 << 30: "5.0G"} {
		if got := humanSize(n); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestLsDate(t *testing.T) {
	now := time.Now()
	recent := now.Add(-2 * time.Hour)
	if got := lsDate(recent); got != recent.Format("Jan _2 15:04") {
		t.Errorf("recent: %q", got)
	}
	old := now.AddDate(-1, 0, 0)
	if got := lsDate(old); got != old.Format("Jan _2  2006") {
		t.Errorf("old: %q", got)
	}
}

func fixture(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "small.txt"), []byte("hi"), 0o644)
	os.WriteFile(filepath.Join(d, "big.bin"), make([]byte, 5000), 0o600)
	os.WriteFile(filepath.Join(d, ".hidden"), nil, 0o644)
	os.Mkdir(filepath.Join(d, "sub"), 0o755)
	os.Symlink("small.txt", filepath.Join(d, "link"))
	old := time.Now().Add(-48 * time.Hour)
	os.Chtimes(filepath.Join(d, "big.bin"), old, old)
	return d
}

func list(t *testing.T, args map[string]interface{}) string {
	t.Helper()
	b, _ := json.Marshal(args)
	out, err := ListOfFiles(b)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func names(out string) []string {
	var n []string
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.HasPrefix(l, "total ") {
			continue
		}
		f := strings.Fields(l)
		n = append(n, f[len(f)-1])
		if strings.Contains(l, " -> ") {
			n[len(n)-1] = f[len(f)-3]
		}
	}
	return n
}

func TestListingOptions(t *testing.T) {
	d := fixture(t)

	out := list(t, map[string]interface{}{"path": d})
	if strings.Contains(out, ".hidden") {
		t.Errorf("dotfile shown without all:\n%s", out)
	}
	if !strings.Contains(out, "link -> small.txt") || !strings.HasPrefix(out, "total ") {
		t.Errorf("long listing missing symlink target or total:\n%s", out)
	}
	if strings.Join(names(out), " ") != "big.bin link small.txt sub" {
		t.Errorf("name order: %v", names(out))
	}

	out = list(t, map[string]interface{}{"path": d, "all": true})
	if n := names(out); n[0] != "." || n[1] != ".." || !strings.Contains(out, ".hidden") {
		t.Errorf("all: %v", n)
	}

	out = list(t, map[string]interface{}{"path": d, "sort": "size"})
	if names(out)[0] != "big.bin" {
		t.Errorf("sort size: %v", names(out))
	}
	out = list(t, map[string]interface{}{"path": d, "sort": "time", "reverse": true})
	if names(out)[0] != "big.bin" { // oldest first when reversed
		t.Errorf("sort time reverse: %v", names(out))
	}
	out = list(t, map[string]interface{}{"path": d, "dirs_first": true})
	if names(out)[0] != "sub" {
		t.Errorf("dirs_first: %v", names(out))
	}

	short := list(t, map[string]interface{}{"path": d, "long": false})
	if short != "big.bin\nlink\nsmall.txt\nsub/\n" {
		t.Errorf("long=false: %q", short)
	}

	var entries []Entry
	json.Unmarshal([]byte(list(t, map[string]interface{}{"path": d, "output_format": "json"})), &entries)
	byName := map[string]Entry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	if e := byName["link"]; e.Type != "symlink" || e.Target != "small.txt" || e.Mode[0] != 'l' {
		t.Errorf("symlink entry: %+v (must be lstat'ed, not followed)", e)
	}
	if e := byName["big.bin"]; e.ModeOctal != "0600" || e.Size != 5000 || e.Mode != "-rw-------" {
		t.Errorf("big.bin entry: %+v", e)
	}

	b, _ := json.Marshal(map[string]interface{}{"path": d, "sort": "bogus"})
	if _, err := ListOfFiles(b); err == nil {
		t.Error("bad sort accepted")
	}
}

// TestMatchesGNULs compares every entry's columns with GNU ls -la on the
// same directory. Runs wherever GNU coreutils ls is installed (Linux CI).
func TestMatchesGNULs(t *testing.T) {
	if out, err := exec.Command("ls", "--version").Output(); err != nil || !strings.Contains(string(out), "GNU") {
		t.Skip("GNU ls not available")
	}
	d := fixture(t)
	os.Chmod(filepath.Join(d, "sub"), fs.ModeSticky|0o1777)
	os.WriteFile(filepath.Join(d, "suid"), nil, 0o755)
	os.Chmod(filepath.Join(d, "suid"), fs.ModeSetuid|0o755)

	cmd := exec.Command("ls", "-la", d)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "TZ="+os.Getenv("TZ"))
	want, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	got := list(t, map[string]interface{}{"path": d, "all": true})

	lines := func(s string) map[string]string {
		m := map[string]string{}
		for _, l := range strings.Split(strings.TrimSpace(s), "\n") {
			f := strings.Fields(l)
			if f[0] == "total" {
				m["total"] = f[1]
				continue
			}
			nameAt := 8
			m[f[nameAt]] = strings.Join(f, " ")
		}
		return m
	}
	w, g := lines(string(want)), lines(got)
	if len(w) != len(g) {
		t.Fatalf("entry count: ls %d, ours %d\nls:\n%s\nours:\n%s", len(w), len(g), want, got)
	}
	for name, wl := range w {
		if g[name] != wl {
			t.Errorf("%s:\n  ls:   %s\n  ours: %s", name, wl, g[name])
		}
	}
}
