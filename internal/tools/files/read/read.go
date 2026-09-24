package readfile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon/internal/fsafe"
)

// ReadFileArgs defines the parameters for the files/read tool.
type ReadFileArgs struct {
	Path      string `json:"path"`                 // Path is the absolute path to the file to read. Required.
	Offset    *int64 `json:"offset,omitempty"`     // Offset is the starting byte offset. Takes precedence if limit is set and lines are not.
	Limit     *int64 `json:"limit,omitempty"`      // Limit is the maximum number of bytes to read. Defaults to 10KB safely.
	StartLine *int   `json:"start_line,omitempty"` // StartLine is the starting line number (1-indexed). Takes precedence over byte offsets.
	EndLine   *int   `json:"end_line,omitempty"`   // EndLine is the inclusive ending line number.
	// NoFollow is set by the daemon (never the caller) when this runs as
	// root under a paths: restriction: a symlink in any path component is
	// then refused instead of followed out of the allowed directories.
	NoFollow bool `json:"_no_follow,omitempty"`
}

const maxSafeBytes = 10240 // 10KB fallback

func Read(argsJSON []byte) (string, error) {
	var args ReadFileArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	var file *os.File
	var err error
	if args.NoFollow {
		file, err = fsafe.OpenFile(args.Path, os.O_RDONLY)
	} else {
		file, err = os.Open(args.Path)
	}
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Detect if file is binary
	{
		sniffBuf := make([]byte, 512)
		n, err := file.Read(sniffBuf)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("failed to read file for type detection: %v", err)
		}
		if n > 0 {
			contentType := http.DetectContentType(sniffBuf[:n])
			if !strings.HasPrefix(contentType, "text/") && contentType != "application/json" {
				// verify with a null byte check, as some valid config files might be misidentified
				isBinary := false
				for _, b := range sniffBuf[:n] {
					if b == 0 {
						isBinary = true
						break
					}
				}
				if isBinary {
					return "", fmt.Errorf("cannot read binary file (detected type: %s)", contentType)
				}
			}
			// Reset file pointer
			if _, err := file.Seek(0, io.SeekStart); err != nil {
				return "", fmt.Errorf("failed to reset file pointer: %v", err)
			}
		}
	}

	// 1. Line-level extraction takes precedence
	if args.StartLine != nil || args.EndLine != nil {
		start := 1
		if args.StartLine != nil && *args.StartLine > 0 {
			start = *args.StartLine
		}
		end := -1
		if args.EndLine != nil && *args.EndLine >= start {
			end = *args.EndLine
		}

		scanner := bufio.NewScanner(file)
		var result string
		lineNum := 1
		for scanner.Scan() {
			if lineNum >= start {
				if end != -1 && lineNum > end {
					break
				}
				result += scanner.Text() + "\n"
			}
			lineNum++
		}
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("error reading lines: %v", err)
		}
		return result, nil
	}

	// 2. Byte-level extraction
	if args.Offset != nil || args.Limit != nil {
		offset := int64(0)
		if args.Offset != nil && *args.Offset > 0 {
			offset = *args.Offset
		}

		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return "", fmt.Errorf("failed to seek: %v", err)
		}

		var buf []byte
		if args.Limit != nil && *args.Limit > 0 {
			// If limit is explicitly requested, allow it (no 10KB cap, user knows what they're doing)
			buf = make([]byte, *args.Limit)
			n, err := io.ReadFull(file, buf)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return "", fmt.Errorf("read error: %v", err)
			}
			return string(buf[:n]), nil
		}

		// If limit is nil but offset is provided, read remaining file (up to safe max)
		buf = make([]byte, maxSafeBytes)
		n, err := io.ReadFull(file, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return "", fmt.Errorf("read error: %v", err)
		}
		contentStr := string(buf[:n])
		if n == maxSafeBytes {
			contentStr += "\n\n[WARNING: File truncated at safe context limit. Pass explicit limit to bypass.]"
		}
		return contentStr, nil
	}

	// 3. Safe fallback (no args provided)
	buf := make([]byte, maxSafeBytes)
	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", fmt.Errorf("read error: %v", err)
	}
	contentStr := string(buf[:n])
	if n == maxSafeBytes {
		contentStr += "\n\n[WARNING: File truncated at safe context limit. Use start_line/end_line or offset/limit for full file inspection.]"
	}
	return contentStr, nil
}
