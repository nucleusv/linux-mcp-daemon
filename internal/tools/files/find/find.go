package findfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// FindFileArgs defines the parameters for the files/find tool.
type FindFileArgs struct {
	Path         string `json:"path"`                    // Path is the starting directory for the search. Defaults to "/".
	Name         string `json:"name,omitempty"`          // Name is the glob pattern to match filenames.
	Type         string `json:"type,omitempty"`          // Type filters by file type ('f' for file, 'd' for directory, 'l' for symlink).
	Mtime        string `json:"mtime,omitempty"`         // Mtime filters by modification time (e.g. "+7" for older than 7 days).
	Size         string `json:"size,omitempty"`          // Size filters by file size (e.g. "+100M" for larger than 100MB).
	MaxDepth     int    `json:"max_depth,omitempty"`     // MaxDepth restricts the depth of the search.
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format. Defaults to text.
}

func Find(argsJSON []byte) (string, error) {
	var args FindFileArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	searchPath := args.Path
	if searchPath == "" {
		searchPath = "/" // Safe fallback but usually bad for performance, maybe users should be explicit
	}

	cmdArgs := []string{searchPath}

	if args.MaxDepth > 0 {
		cmdArgs = append(cmdArgs, "-maxdepth", fmt.Sprintf("%d", args.MaxDepth))
	}

	if args.Name != "" {
		cmdArgs = append(cmdArgs, "-name", args.Name)
	}

	if args.Type != "" {
		cmdArgs = append(cmdArgs, "-type", args.Type)
	}

	if args.Mtime != "" {
		cmdArgs = append(cmdArgs, "-mtime", args.Mtime)
	}

	if args.Size != "" {
		cmdArgs = append(cmdArgs, "-size", args.Size)
	}

	cmd := exec.Command("find", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// find often exits with 1 if permission is denied on some directories, 
		// but still returns valid output for others.
		if len(stdout.Bytes()) == 0 {
			return "", fmt.Errorf("find error: %v, stderr: %s", err, stderr.String())
		}
	}

	return stdout.String(), nil
}
