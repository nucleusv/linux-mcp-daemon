// Package filetype determines a file's MIME type natively - the answer
// `file -b --mime-type` gives for the common cases - without running the
// external `file` binary (which minimal hosts often lack).
package filetype

import (
	"bytes"
	"debug/elf"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// FileTypeArgs defines the parameters for files/filetype and the
// file://{path}/type resource.
type FileTypeArgs struct {
	Path string `json:"path"` // Path is the absolute path to the file. Required.
}

// sniffLen is how much of a file is read to identify it. Every signature
// below lies within it (tar's "ustar" is at offset 257).
const sniffLen = 8192

func Type(argsJSON []byte) (string, error) {
	var args FileTypeArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("path argument is required")
	}
	mime, err := Detect(args.Path)
	if err != nil {
		return "", err
	}
	return mime + "\n", nil
}

// Detect returns path's MIME type. Like `file`, it describes a symlink
// itself rather than following it.
func Detect(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	switch mode := info.Mode(); {
	case mode&os.ModeSymlink != 0:
		return "inode/symlink", nil
	case mode.IsDir():
		return "inode/directory", nil
	case mode&os.ModeCharDevice != 0:
		return "inode/chardevice", nil
	case mode&os.ModeDevice != 0:
		return "inode/blockdevice", nil
	case mode&os.ModeNamedPipe != 0:
		return "inode/fifo", nil
	case mode&os.ModeSocket != 0:
		return "inode/socket", nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	head := make([]byte, sniffLen)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", err
	}
	head = head[:n]
	if n == 0 {
		return "inode/x-empty", nil
	}
	if bytes.HasPrefix(head, []byte("\x7fELF")) {
		return elfType(f), nil
	}
	return sniff(head, filepath.Base(path)), nil
}

// magic lists signatures http.DetectContentType doesn't know, or names
// differently from `file`.
var magic = []struct {
	offset int
	sig    string
	mime   string
}{
	{0, "SQLite format 3\x00", "application/vnd.sqlite3"},
	{0, "!<arch>\ndebian", "application/vnd.debian.binary-package"},
	{0, "!<arch>\n", "application/x-archive"},
	{0, "\xed\xab\xee\xdb", "application/x-rpm"},
	{0, "\x1f\x8b", "application/gzip"},
	{0, "BZh", "application/x-bzip2"},
	{0, "\xfd7zXZ\x00", "application/x-xz"},
	{0, "\x28\xb5\x2f\xfd", "application/zstd"},
	{0, "7z\xbc\xaf\x27\x1c", "application/x-7z-compressed"},
	{0, "\x04\x22\x4d\x18", "application/x-lz4"},
	{257, "ustar", "application/x-tar"},
	{0, "%PDF-", "application/pdf"},
	{0, "\x89PNG\r\n\x1a\n", "image/png"},
	{0, "\xff\xd8\xff", "image/jpeg"},
	{0, "GIF87a", "image/gif"},
	{0, "GIF89a", "image/gif"},
	{0, "-----BEGIN PGP", "application/pgp-keys"},
	{0, "-----BEGIN ", "text/plain"}, // PEM certificates and keys
	{0, "\xca\xfe\xba\xbe", "application/x-java-applet"},
	{0, "\x00asm", "application/wasm"},
}

// interpreters maps a #! interpreter to the type `file` reports.
var interpreters = map[string]string{
	"sh": "text/x-shellscript", "bash": "text/x-shellscript", "dash": "text/x-shellscript",
	"zsh": "text/x-shellscript", "ksh": "text/x-shellscript",
	"python": "text/x-script.python", "python2": "text/x-script.python", "python3": "text/x-script.python",
	"perl": "text/x-perl", "ruby": "text/x-ruby", "node": "application/javascript",
	"php": "text/x-php", "awk": "text/x-awk", "lua": "text/x-lua",
}

func sniff(head []byte, name string) string {
	for _, m := range magic {
		if len(head) >= m.offset+len(m.sig) && string(head[m.offset:m.offset+len(m.sig)]) == m.sig {
			return m.mime
		}
	}
	if bytes.HasPrefix(head, []byte("#!")) {
		if t := scriptType(head); t != "" {
			return t
		}
	}

	// Formats recognized by the standard library's sniffer (images, audio,
	// video, fonts, zip, html, xml, ...), except its catch-all answers.
	switch ct := strings.SplitN(http.DetectContentType(head), ";", 2)[0]; ct {
	case "text/plain", "application/octet-stream":
	case "text/xml":
		return "text/xml"
	case "application/zip":
		return "application/zip"
	default:
		return ct
	}

	if !isText(head) {
		return "application/octet-stream"
	}
	trimmed := bytes.TrimSpace(head)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') && len(head) < sniffLen && json.Valid(trimmed) {
		return "application/json"
	}
	if strings.HasSuffix(name, ".sh") || strings.HasSuffix(name, ".bash") {
		return "text/x-shellscript"
	}
	return "text/plain"
}

// scriptType reads a "#!/usr/bin/env python3"-style first line.
func scriptType(head []byte) string {
	line := string(head[2:])
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	interp := filepath.Base(fields[0])
	if interp == "env" {
		for _, f := range fields[1:] {
			if !strings.HasPrefix(f, "-") {
				interp = f
				break
			}
		}
	}
	// python3.12 -> python3, perl5.36 -> perl5 -> perl
	for {
		if t, ok := interpreters[interp]; ok {
			return t
		}
		trimmed := strings.TrimRight(interp, "0123456789.")
		if trimmed == interp || trimmed == "" {
			break
		}
		interp = trimmed
	}
	return "text/plain"
}

// isText: no NUL bytes, and valid UTF-8 (a multi-byte rune cut off by the
// end of the sample doesn't count against it).
func isText(b []byte) bool {
	if bytes.IndexByte(b, 0) >= 0 {
		return false
	}
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		if r == utf8.RuneError && size == 1 {
			return len(b) < utf8.UTFMax && !utf8.FullRune(b)
		}
		b = b[size:]
	}
	return true
}

// elfType tells executables, position-independent executables and shared
// libraries apart the way `file` does.
func elfType(f *os.File) string {
	ef, err := elf.NewFile(f)
	if err != nil {
		return "application/octet-stream"
	}
	defer ef.Close()
	switch ef.Type {
	case elf.ET_EXEC:
		return "application/x-executable"
	case elf.ET_REL:
		return "application/x-object"
	case elf.ET_CORE:
		return "application/x-coredump"
	case elf.ET_DYN:
		// A PIE executable is ET_DYN too; like file(1), tell it apart by
		// the DF_1_PIE flag alone - libc.so.6 has a program interpreter (it
		// can be run) but is still a shared library.
		if flags, err := ef.DynValue(elf.DT_FLAGS_1); err == nil {
			for _, v := range flags {
				if elf.DynFlag1(v)&elf.DF_1_PIE != 0 {
					return "application/x-pie-executable"
				}
			}
		}
		return "application/x-sharedlib"
	}
	return "application/octet-stream"
}
