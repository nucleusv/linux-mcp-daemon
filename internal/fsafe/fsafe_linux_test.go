//go:build linux

package fsafe

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// layout: dir/allowed/{real.txt, sub/}, dir/secret.txt outside, and
// symlinks inside "allowed" pointing out of it.
func layout(t *testing.T) (allowed, secret string) {
	t.Helper()
	dir := t.TempDir()
	allowed = filepath.Join(dir, "allowed")
	secret = filepath.Join(dir, "secret.txt")
	must(t, os.MkdirAll(filepath.Join(allowed, "sub"), 0755))
	must(t, os.WriteFile(filepath.Join(allowed, "real.txt"), []byte("ok\n"), 0644))
	must(t, os.WriteFile(secret, []byte("SECRET\n"), 0600))
	must(t, os.Symlink(secret, filepath.Join(allowed, "file-link")))
	must(t, os.Symlink(dir, filepath.Join(allowed, "dir-link")))
	return allowed, secret
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpenFileRefusesSymlinks(t *testing.T) {
	allowed, _ := layout(t)
	f, err := OpenFile(filepath.Join(allowed, "real.txt"), os.O_RDONLY)
	must(t, err)
	b, _ := io.ReadAll(f)
	f.Close()
	if string(b) != "ok\n" {
		t.Errorf("read %q", b)
	}
	for _, p := range []string{"file-link", "dir-link/secret.txt"} {
		if _, err := OpenFile(filepath.Join(allowed, p), os.O_RDONLY); !errors.Is(err, ErrSymlink) {
			t.Errorf("%s: want ErrSymlink, got %v", p, err)
		}
	}
}

func TestCreateFileRefusesSymlinks(t *testing.T) {
	allowed, secret := layout(t)
	for _, p := range []string{"file-link", "dir-link/secret.txt", "dir-link/new.txt"} {
		_, err := CreateFile(filepath.Join(allowed, p), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644, true)
		if !errors.Is(err, ErrSymlink) {
			t.Errorf("%s: want ErrSymlink, got %v", p, err)
		}
	}
	if b, _ := os.ReadFile(secret); string(b) != "SECRET\n" {
		t.Errorf("secret was written through a symlink: %q", b)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(allowed), "new.txt")); err == nil {
		t.Error("a file was created outside through dir-link")
	}

	// Normal use: new file in new directories.
	f, err := CreateFile(filepath.Join(allowed, "a/b/new.txt"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640, true)
	must(t, err)
	f.WriteString("x")
	f.Close()
	if b, _ := os.ReadFile(filepath.Join(allowed, "a/b/new.txt")); string(b) != "x" {
		t.Errorf("created file has %q", b)
	}
}

func TestMkdirAllRefusesSymlinks(t *testing.T) {
	allowed, _ := layout(t)
	if _, err := MkdirAll(filepath.Join(allowed, "dir-link/escape"), 0755); !errors.Is(err, ErrSymlink) {
		t.Errorf("want ErrSymlink, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(allowed), "escape")); err == nil {
		t.Error("a directory was created outside through dir-link")
	}
}

func TestChdir(t *testing.T) {
	allowed, _ := layout(t)
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	n, err := Open(filepath.Join(allowed, "sub"))
	must(t, err)
	defer n.Close()
	must(t, n.Chdir())
	cwd, _ := os.Getwd()
	if cwd != filepath.Join(allowed, "sub") {
		t.Errorf("cwd %s", cwd)
	}
}
