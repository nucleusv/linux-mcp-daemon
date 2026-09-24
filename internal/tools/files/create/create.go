package createfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/fsafe"
)

// CreateFileArgs defines the parameters for the files/create tool.
type CreateFileArgs struct {
	Path    string `json:"path"`              // Path is the absolute path to the file to create. Required.
	Content string `json:"content,omitempty"` // Content is the data to write to the file. If omitted, creates an empty file or updates mtime.
	// NoFollow is set by the daemon (never the caller) when this runs as
	// root under a paths: restriction: a symlink in any path component is
	// then refused instead of followed out of the allowed directories.
	NoFollow bool `json:"_no_follow,omitempty"`
}

func Create(argsJSON []byte) (string, error) {
	var args CreateFileArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	if args.NoFollow {
		return createNoFollow(args)
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

// createNoFollow is Create without following symlinks: parent directories
// are created and the file opened relative to verified directories, so a
// symlink planted anywhere on the path is refused, never written through.
func createNoFollow(args CreateFileArgs) (string, error) {
	flag := os.O_WRONLY | os.O_CREATE
	if args.Content != "" {
		flag |= os.O_TRUNC
	}
	f, err := fsafe.CreateFile(args.Path, flag, 0644, true)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()
	if args.Content == "" {
		now := time.Now()
		// The descriptor's /proc link is this exact inode.
		if err := os.Chtimes(fmt.Sprintf("/proc/self/fd/%d", f.Fd()), now, now); err != nil {
			return "", fmt.Errorf("failed to touch file: %v", err)
		}
		return fmt.Sprintf("Successfully touched %s", args.Path), nil
	}
	if _, err := f.WriteString(args.Content); err != nil {
		return "", fmt.Errorf("failed to write to file: %v", err)
	}
	return fmt.Sprintf("Successfully created and wrote to %s", args.Path), nil
}
