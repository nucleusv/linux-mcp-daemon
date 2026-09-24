//go:build linux

// These tests check that every file tool, given _no_follow (as the daemon
// sets for a root call under a paths: restriction), refuses to follow a
// symlink out of the allowed directory - and still works normally.
package fsafe_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	disk_usage "github.com/nucleusv/linux-mcp-daemon/internal/tools/disks/usage"
	createfile "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/create"
	"github.com/nucleusv/linux-mcp-daemon/internal/tools/files/filetype"
	findfile "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/find"
	listfiles "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/list"
	readfile "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/read"
	updatefile "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/update"
)

func setup(t *testing.T) (allowed, outside string) {
	t.Helper()
	dir := t.TempDir()
	allowed = filepath.Join(dir, "allowed")
	outside = filepath.Join(dir, "outside")
	for _, d := range []string{allowed, outside} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("SECRET-LINE\n"), 0600)
	os.WriteFile(filepath.Join(allowed, "notes.txt"), []byte("one\ntwo\n"), 0644)
	os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(allowed, "file-link"))
	os.Symlink(outside, filepath.Join(allowed, "dir-link"))
	return allowed, outside
}

func call(fn func([]byte) (string, error), args map[string]any) (string, error) {
	args["_no_follow"] = true
	b, _ := json.Marshal(args)
	return fn(b)
}

func TestNoFollowRefusesEscapes(t *testing.T) {
	allowed, outside := setup(t)
	secret := filepath.Join(outside, "secret.txt")

	// Control: without _no_follow the same read does escape - so the
	// refusals below are the fix at work, not a broken test setup.
	b, _ := json.Marshal(map[string]any{"path": allowed + "/dir-link/secret.txt"})
	if out, err := readfile.Read(b); err != nil || !strings.Contains(out, "SECRET") {
		t.Fatalf("control read didn't escape (%q, %v) - test setup is wrong", out, err)
	}

	attacks := []struct {
		name string
		fn   func([]byte) (string, error)
		args map[string]any
	}{
		{"read file-link", readfile.Read, map[string]any{"path": allowed + "/file-link"}},
		{"read via dir-link", readfile.Read, map[string]any{"path": allowed + "/dir-link/secret.txt"}},
		{"create over file-link", createfile.Create, map[string]any{"path": allowed + "/file-link", "content": "PWNED"}},
		{"create via dir-link", createfile.Create, map[string]any{"path": allowed + "/dir-link/new.txt", "content": "PWNED"}},
		{"append to file-link", updatefile.Update, map[string]any{"path": allowed + "/file-link", "content": "PWNED", "append": true}},
		{"replace in file-link", updatefile.Update, map[string]any{"path": allowed + "/file-link", "content": "PWNED", "start_line": 1, "end_line": 1}},
		{"list via dir-link", listfiles.ListOfFiles, map[string]any{"path": allowed + "/dir-link"}},
		{"find via dir-link", findfile.Find, map[string]any{"path": allowed + "/dir-link"}},
		{"usage via dir-link", disk_usage.Usage, map[string]any{"path": allowed + "/dir-link"}},
		{"filetype via dir-link", filetype.Type, map[string]any{"path": allowed + "/dir-link/secret.txt"}},
	}
	wd, _ := os.Getwd()
	for _, a := range attacks {
		out, err := call(a.fn, a.args)
		os.Chdir(wd) // find changes directory
		if err == nil || strings.Contains(out, "SECRET") || strings.Contains(out, "secret.txt") {
			t.Errorf("%s: not refused: out=%q err=%v", a.name, out, err)
		}
	}
	if b, _ := os.ReadFile(secret); string(b) != "SECRET-LINE\n" {
		t.Errorf("the file outside was modified: %q", b)
	}
	if _, err := os.Stat(filepath.Join(outside, "new.txt")); err == nil {
		t.Error("a file was created outside")
	}
	// The final symlink itself is described, not followed.
	if out, err := call(filetype.Type, map[string]any{"path": allowed + "/file-link"}); err != nil || out != "inode/symlink\n" {
		t.Errorf("filetype of the link: %q %v", out, err)
	}
}

func TestNoFollowNormalUse(t *testing.T) {
	allowed, _ := setup(t)
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	p := allowed + "/notes.txt"
	if out, err := call(readfile.Read, map[string]any{"path": p}); err != nil || !strings.Contains(out, "two") {
		t.Errorf("read: %q %v", out, err)
	}
	if _, err := call(updatefile.Update, map[string]any{"path": p, "content": "TWO", "start_line": 2, "end_line": 2}); err != nil {
		t.Errorf("update: %v", err)
	}
	if _, err := call(updatefile.Update, map[string]any{"path": p, "content": "three\n", "append": true}); err != nil {
		t.Errorf("append: %v", err)
	}
	if b, _ := os.ReadFile(p); string(b) != "one\nTWO\nthree\n" {
		t.Errorf("after update: %q", b)
	}
	if _, err := call(createfile.Create, map[string]any{"path": allowed + "/new/dir/f.txt", "content": "hi"}); err != nil {
		t.Errorf("create: %v", err)
	}
	if _, err := call(createfile.Create, map[string]any{"path": allowed + "/touched"}); err != nil {
		t.Errorf("touch: %v", err)
	}
	if out, err := call(listfiles.ListOfFiles, map[string]any{"path": allowed, "all": true}); err != nil || !strings.Contains(out, "notes.txt") || !strings.Contains(out, "file-link ->") {
		t.Errorf("list: %q %v", out, err)
	}
	out, err := call(findfile.Find, map[string]any{"path": allowed, "name": "*.txt"})
	os.Chdir(wd)
	if err != nil || !strings.Contains(out, allowed+"/notes.txt") || !strings.Contains(out, allowed+"/new/dir/f.txt") || strings.Contains(out, "secret") {
		t.Errorf("find: %q %v", out, err)
	}
	if out, err := call(filetype.Type, map[string]any{"path": p}); err != nil || out != "text/plain\n" {
		t.Errorf("filetype: %q %v", out, err)
	}
	if out, err := call(disk_usage.Usage, map[string]any{"path": allowed}); err != nil || !strings.Contains(out, "Total size of "+allowed) {
		t.Errorf("usage: %q %v", out, err)
	}
}
