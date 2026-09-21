package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type GetDiskUsageArgs struct {
	Path       string `json:"path"`
	MaxDepth   int    `json:"max_depth,omitempty"`
	Privileged bool   `json:"privileged,omitempty"`
}

// GetDiskUsage calculates disk usage by traversing a directory tree (equivalent to du -sh).
func GetDiskUsage(rawArgs json.RawMessage) (string, error) {
	var args GetDiskUsageArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	cleanPath := filepath.Clean(args.Path)
	
	// Ensure the root path exists
	_, err := os.Stat(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to access path %s: %v", cleanPath, err)
	}

	var rootDepth int
	if cleanPath == "/" || cleanPath == "." {
		rootDepth = 0
	} else {
		rootDepth = strings.Count(cleanPath, string(os.PathSeparator))
	}

	totalSize := int64(0)
	dirSizes := make(map[string]int64)

	err = filepath.WalkDir(cleanPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Permission denied errors on files inside the path are common. 
			// We skip them, but they indicate the AI might need 'privileged: true' to get an accurate total.
			return nil
		}

		info, err := d.Info()
		if err == nil && !info.IsDir() {
			size := info.Size()
			totalSize += size
			
			// Accumulate sizes for parent directories up to max_depth
			currentDepth := strings.Count(path, string(os.PathSeparator)) - rootDepth
			
			if args.MaxDepth > 0 && currentDepth <= args.MaxDepth {
				parent := filepath.Dir(path)
				for strings.HasPrefix(parent, cleanPath) {
					dirSizes[parent] += size
					if parent == cleanPath || parent == "/" || parent == "." {
						break
					}
					parent = filepath.Dir(parent)
				}
			}
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error during traversal: %v", err)
	}

	var result string
	
	if args.MaxDepth > 0 {
		result += fmt.Sprintf("Directory sizes (up to depth %d):\n", args.MaxDepth)
		for dir, size := range dirSizes {
			result += fmt.Sprintf("%s\t%s\n", formatBytes(uint64(size)), dir)
		}
		result += "---\n"
	}
	
	result += fmt.Sprintf("Total size of %s: %s\n", cleanPath, formatBytes(uint64(totalSize)))

	return result, nil
}
