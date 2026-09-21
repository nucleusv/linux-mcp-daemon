package createfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CreateFileArgs defines the parameters for the files/create tool.
type CreateFileArgs struct {
	Path    string `json:"path"`              // Path is the absolute path to the file to create. Required.
	Content string `json:"content,omitempty"` // Content is the data to write to the file. If omitted, creates an empty file or updates mtime.
}

func Create(argsJSON []byte) (string, error) {
	var args CreateFileArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	dir := filepath.Dir(args.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create parent directories: %v", err)
	}

	if args.Content == "" {
		// Just touch the file
		file, err := os.OpenFile(args.Path, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return "", fmt.Errorf("failed to touch file: %v", err)
		}
		file.Close()
		currentTime := time.Now()
		os.Chtimes(args.Path, currentTime, currentTime)
		return fmt.Sprintf("Successfully touched %s", args.Path), nil
	}

	// Write content
	if err := os.WriteFile(args.Path, []byte(args.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write to file: %v", err)
	}

	return fmt.Sprintf("Successfully created and wrote to %s", args.Path), nil
}
