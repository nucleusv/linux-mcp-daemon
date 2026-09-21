package readfile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type ReadFileArgs struct {
	Path      string `json:"path"`
	Offset    *int64 `json:"offset,omitempty"`
	Limit     *int64 `json:"limit,omitempty"`
	StartLine *int   `json:"start_line,omitempty"`
	EndLine   *int   `json:"end_line,omitempty"`
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

	file, err := os.Open(args.Path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

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
