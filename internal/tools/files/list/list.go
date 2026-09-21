package listfiles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GetListOfFilesArgs defines the parameters for the list_directory tool.
type GetListOfFilesArgs struct {
	OutputFormat string `json:\"output_format,omitempty\"` // OutputFormat specifies the desired output format (e.g. \"json\"). Defaults to text.
	Path       string `json:"path"`                 // Path is the absolute path of the directory to list.
	Privileged bool   `json:"privileged,omitempty"` // Privileged executes the tool as the root user (if authorized in mcp-sudo.yaml).
}

// ListDirectory reads the contents of the specified directory.
func ListOfFiles(argsJSON []byte) (string, error) {
	var args GetListOfFilesArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
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

	var jsonResult []map[string]interface{}
	var result string
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			result += fmt.Sprintf("[?] %s\n", entry.Name())
			if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
				jsonResult = append(jsonResult, map[string]interface{}{"name": entry.Name(), "error": "failed to stat"})
			}
			continue
		}

		size := info.Size()
		modTime := info.ModTime().Format("2006-01-02 15:04:05")

		if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
			jsonResult = append(jsonResult, map[string]interface{}{
				"name": entry.Name(),
				"size": size,
				"modified": modTime,
				"is_dir": entry.IsDir(),
			})
		}

		if entry.IsDir() {
			result += fmt.Sprintf("[DIR]  %s/ (modified: %s)\n", entry.Name(), modTime)
		} else {
			result += fmt.Sprintf("[FILE] %s (%d bytes, modified: %s)\n", entry.Name(), size, modTime)
		}
	}

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		if len(jsonResult) == 0 {
			return "[]", nil
		}
		b, _ := json.Marshal(jsonResult)
		return string(b), nil
	}

	if result == "" {
		return "Directory is empty.", nil
	}

	return result, nil
}
