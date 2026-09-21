package list_directory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ListDirectoryArgs defines the parameters for the list_directory tool.
type ListDirectoryArgs struct {
	// Path is the absolute path of the directory to list.
	Path       string `json:"path"`
	// Privileged executes the tool as the root user (if authorized in mcp-sudo.yaml).
	Privileged bool   `json:"privileged,omitempty"`
}

// ListDirectory reads the contents of the specified directory.
func ListDirectory(rawArgs json.RawMessage) (string, error) {
	var args ListDirectoryArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	// Basic security: prevent relative path traversal to keep it predictable,
	// though the real security is the file permissions of the worker.
	cleanPath := filepath.Clean(args.Path)

	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %v", err)
	}

	var result string
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			result += fmt.Sprintf("[?] %s\n", entry.Name())
			continue
		}

		size := info.Size()
		modTime := info.ModTime().Format("2006-01-02 15:04:05")
		
		if entry.IsDir() {
			result += fmt.Sprintf("[DIR]  %s/ (modified: %s)\n", entry.Name(), modTime)
		} else {
			result += fmt.Sprintf("[FILE] %s (%d bytes, modified: %s)\n", entry.Name(), size, modTime)
		}
	}

	if result == "" {
		return "Directory is empty.", nil
	}

	return result, nil
}
