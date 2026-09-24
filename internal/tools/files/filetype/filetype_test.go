package filetype

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// write creates a sample file for the detector.
func write(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDetect(t *testing.T) {
	dir := t.TempDir()
	tar := make([]byte, 1024)
	copy(tar, "hello.txt")
	copy(tar[257:], "ustar\x0000")
	cases := map[string]string{
		write(t, dir, "empty", nil):                                           "inode/x-empty",
		write(t, dir, "plain", []byte("hello world\n")):                       "text/plain",
		write(t, dir, "utf8", []byte("привет, мир\n")):                        "text/plain",
		write(t, dir, "sh", []byte("#!/bin/sh\necho hi\n")):                   "text/x-shellscript",
		write(t, dir, "bash", []byte("#!/usr/bin/env bash\necho\n")):          "text/x-shellscript",
		write(t, dir, "py", []byte("#!/usr/bin/env python3\nprint(1)\n")):     "text/x-script.python",
		write(t, dir, "py312", []byte("#!/usr/bin/python3.12\n")):             "text/x-script.python",
		write(t, dir, "json", []byte(`{"a": [1, 2]}`)):                        "application/json",
		write(t, dir, "gz", []byte("\x1f\x8b\x08\x00\x00\x00")):               "application/gzip",
		write(t, dir, "tar", tar):                                             "application/x-tar",
		write(t, dir, "png", []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR")): "image/png",
		write(t, dir, "sqlite", []byte("SQLite format 3\x00rest")):            "application/vnd.sqlite3",
		write(t, dir, "deb", []byte("!<arch>\ndebian-binary   ")):             "application/vnd.debian.binary-package",
		write(t, dir, "bin", []byte{0, 1, 2, 3, 0xff, 0xfe}):                  "application/octet-stream",
		write(t, dir, "pem", []byte("-----BEGIN CERTIFICATE-----\nMIIB\n")):   "text/plain",
		dir: "inode/directory",
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(filepath.Join(dir, "plain"), link); err != nil {
		t.Fatal(err)
	}
	cases[link] = "inode/symlink"
	for p, want := range cases {
		got, err := Detect(p)
		if err != nil || got != want {
			t.Errorf("%s: got %q (%v), want %q", filepath.Base(p), got, err, want)
		}
	}
	if _, err := Detect(filepath.Join(dir, "missing")); err == nil {
		t.Error("missing file: no error")
	}
}

// TestMatchesFileCommand compares against `file -b --mime-type` on real
// system files, where the external command is installed.
func TestMatchesFileCommand(t *testing.T) {
	if _, err := exec.LookPath("file"); err != nil {
		t.Skip("file(1) not installed")
	}
	var paths []string
	for _, pattern := range []string{
		"/bin/sh", "/bin/ls", "/usr/bin/*", "/usr/lib/*/libc.so.6", "/usr/lib/*/*.so.*",
		"/etc/passwd", "/etc/hostname", "/etc/ssl/certs/*.pem", "/usr/share/doc/*/copyright",
		"/usr/share/doc/*/*.gz", "/var/lib/dpkg/status", "/dev/null", "/tmp", "/proc/self/exe",
	} {
		m, _ := filepath.Glob(pattern)
		if len(m) > 40 {
			m = m[:40]
		}
		paths = append(paths, m...)
	}
	mismatches := 0
	for _, p := range paths {
		out, err := exec.Command("file", "-b", "--mime-type", "--", p).Output()
		if err != nil {
			continue
		}
		want := strings.TrimSpace(string(out))
		got, err := Detect(p)
		if err != nil {
			continue
		}
		if got != want {
			mismatches++
			t.Errorf("%s: got %q, file(1) says %q", p, got, want)
		}
	}
	t.Logf("compared %d files, %d mismatches", len(paths), mismatches)
}
