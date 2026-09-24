package updatefile

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon/internal/fsafe"
)

// UpdateFileArgs defines the parameters for the files/update tool.
type UpdateFileArgs struct {
	Path      string `json:"path"`                 // Path is the absolute path to the file to update. Required.
	Content   string `json:"content"`              // Content is the text to append or insert. Required.
	Append    bool   `json:"append,omitempty"`     // Append adds the content to the very end of the file.
	StartLine *int   `json:"start_line,omitempty"` // StartLine specifies the beginning of the line range to replace (1-indexed).
	EndLine   *int   `json:"end_line,omitempty"`   // EndLine specifies the end of the line range to replace.
	// NoFollow is set by the daemon (never the caller) when this runs as
	// root under a paths: restriction: a symlink in any path component is
	// then refused instead of followed out of the allowed directories.
	NoFollow bool `json:"_no_follow,omitempty"`
}

func Update(argsJSON []byte) (string, error) {
	var args UpdateFileArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	if args.Append {
		var file *os.File
		var err error
		if args.NoFollow {
			file, err = fsafe.CreateFile(args.Path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644, false)
		} else {
			file, err = os.OpenFile(args.Path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		}
		if err != nil {
			return "", fmt.Errorf("failed to open file for append: %v", err)
		}
		defer file.Close()

		if _, err := file.WriteString(args.Content); err != nil {
			return "", fmt.Errorf("failed to append to file: %v", err)
		}
		return fmt.Sprintf("Successfully appended to %s", args.Path), nil
	}

	// Line-based replacement
	if args.StartLine == nil || args.EndLine == nil {
		return "", fmt.Errorf("must specify either 'append' or both 'start_line' and 'end_line'")
	}

	if *args.StartLine < 1 || *args.EndLine < *args.StartLine {
		return "", fmt.Errorf("invalid line range: %d to %d", *args.StartLine, *args.EndLine)
	}

	// Read and rewrite through one descriptor: the file written is the
	// file read, and with NoFollow it was reached without any symlink.
	var f *os.File
	var err error
	if args.NoFollow {
		f, err = fsafe.OpenFile(args.Path, os.O_RDWR)
	} else {
		f, err = os.OpenFile(args.Path, os.O_RDWR, 0)
	}
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// Split on "\n" directly rather than with bufio.Scanner, which has a
	// 64KB line limit and can't tell whether the file ended with a newline -
	// dropping that final newline glued the next append onto the last line.
	content := string(data)
	hadTrailingNewline := strings.HasSuffix(content, "\n")
	content = strings.TrimSuffix(content, "\n")
	var lines []string
	if len(data) > 0 {
		lines = strings.Split(content, "\n")
	}

	startIdx := *args.StartLine - 1
	endIdx := *args.EndLine - 1

	if startIdx >= len(lines) {
		// Just append at the end if the start line is beyond file
		lines = append(lines, strings.TrimSuffix(args.Content, "\n"))
	} else {
		if endIdx >= len(lines) {
			endIdx = len(lines) - 1
		}

		var newLines []string
		newLines = append(newLines, lines[:startIdx]...)

		// Split the new content by lines and append. A single trailing
		// newline in the replacement just terminates its last line.
		newContentLines := strings.Split(strings.TrimSuffix(args.Content, "\n"), "\n")
		newLines = append(newLines, newContentLines...)

		if endIdx+1 < len(lines) {
			newLines = append(newLines, lines[endIdx+1:]...)
		}
		lines = newLines
	}

	output := strings.Join(lines, "\n")
	if hadTrailingNewline || len(data) == 0 {
		output += "\n"
	}
	// Rewritten in place, so the file keeps its permissions and owner.
	if err := f.Truncate(0); err != nil {
		return "", fmt.Errorf("failed to write updated file: %v", err)
	}
	if _, err := f.WriteAt([]byte(output), 0); err != nil {
		return "", fmt.Errorf("failed to write updated file: %v", err)
	}

	return fmt.Sprintf("Successfully updated lines %d-%d in %s", *args.StartLine, *args.EndLine, args.Path), nil
}
