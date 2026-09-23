//go:build linux

package chown

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func ownerOf(t *testing.T, p string) (uint32, uint32) {
	t.Helper()
	var st syscall.Stat_t
	if err := syscall.Lstat(p, &st); err != nil {
		t.Fatal(err)
	}
	return st.Uid, st.Gid
}

func TestChownNeverFollowsSymlinks(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("needs root to change ownership")
	}
	base := t.TempDir()
	outside := filepath.Join(base, "outside")
	os.WriteFile(outside, nil, 0o600)
	tree := filepath.Join(base, "tree")
	os.Mkdir(tree, 0o755)
	os.WriteFile(filepath.Join(tree, "file"), nil, 0o600)
	os.Symlink(outside, filepath.Join(tree, "link"))

	call := func(args map[string]interface{}) (string, error) {
		b, _ := json.Marshal(args)
		return Chown(b)
	}
	if _, err := call(map[string]interface{}{"path": filepath.Join(tree, "link"), "owner": "65534:65534"}); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Errorf("symlink target not refused: %v", err)
	}
	out, err := call(map[string]interface{}{"path": tree, "owner": "65534:65534", "recursive": true})
	if err != nil {
		t.Fatal(err)
	}
	if u, g := ownerOf(t, filepath.Join(tree, "file")); u != 65534 || g != 65534 {
		t.Errorf("in-tree file owner %d:%d\n%s", u, g, out)
	}
	if u, _ := ownerOf(t, outside); u != 0 {
		t.Errorf("recursive chown followed a symlink out of the tree - outside file owned by %d", u)
	}
	if !strings.Contains(out, "skipped 1 symlink") {
		t.Errorf("skipped symlink not reported:\n%s", out)
	}
	// group only
	if out, err := call(map[string]interface{}{"path": filepath.Join(tree, "file"), "owner": ":0"}); err != nil || !strings.Contains(out, "-> ") {
		t.Errorf("group-only chown: %q %v", out, err)
	}
	if u, g := ownerOf(t, filepath.Join(tree, "file")); u != 65534 || g != 0 {
		t.Errorf("group-only changed owner too: %d:%d", u, g)
	}
}
