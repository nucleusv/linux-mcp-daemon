package updatefile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// UpdateFileArgs defines the parameters for the files/update tool.
type UpdateFileArgs struct {
	Path      string `json:"path"`                 // Path is the absolute path to the file to update. Required.
	Content   string `json:"content"`              // Content is the text to append or insert. Required.
	Append    bool   `json:"append,omitempty"`     // Append adds the content to the very end of the file.
	StartLine *int   `json:"start_line,omitempty"` // StartLine specifies the beginning of the line range to replace (1-indexed).
	EndLine   *int   `json:"end_line,omitempty"`   // EndLine specifies the end of the line range to replace.
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
		file, err := os.OpenFile(args.Path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
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

	file, err := os.Open(args.Path)
	if err != nil {
		return "", fmt.Errorf("failed to open file for reading: %v", err)
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file lines: %v", err)
	}

	startIdx := *args.StartLine - 1
	endIdx := *args.EndLine - 1

	if startIdx >= len(lines) {
		// Just append at the end if the start line is beyond file
		lines = append(lines, args.Content)
	} else {
		if endIdx >= len(lines) {
			endIdx = len(lines) - 1
		}
		
		var newLines []string
		newLines = append(newLines, lines[:startIdx]...)
		
		// Split the new content by lines and append
		newContentLines := strings.Split(args.Content, "\n")
		newLines = append(newLines, newContentLines...)
		
		if endIdx+1 < len(lines) {
			newLines = append(newLines, lines[endIdx+1:]...)
		}
		lines = newLines
	}

	output := strings.Join(lines, "\n")
	if err := os.WriteFile(args.Path, []byte(output), 0644); err != nil {
		return "", fmt.Errorf("failed to write updated file: %v", err)
	}

	return fmt.Sprintf("Successfully updated lines %d-%d in %s", *args.StartLine, *args.EndLine, args.Path), nil
}
