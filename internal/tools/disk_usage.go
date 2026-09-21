package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type GetDiskUsageArgs struct {
	Path          string   `json:"path"`
	MaxDepth      int      `json:"max_depth,omitempty"`
	OneFileSystem bool     `json:"one_file_system,omitempty"`
	Exclude       []string `json:"exclude,omitempty"`
	All           bool     `json:"all,omitempty"`
	Privileged    bool     `json:"privileged,omitempty"`
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
	
	// Ensure the root path exists and get root device ID
	rootInfo, err := os.Stat(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to access path %s: %v", cleanPath, err)
	}

	var rootDev uint64
	if stat, ok := rootInfo.Sys().(*syscall.Stat_t); ok {
		rootDev = stat.Dev
	}

	var rootDepth int
	if cleanPath == "/" || cleanPath == "." {
		rootDepth = 0
	} else {
		rootDepth = strings.Count(cleanPath, string(os.PathSeparator))
	}

	totalSize := int64(0)
	dirSizes := make(map[string]int64)
	var allFiles string

	err = filepath.WalkDir(cleanPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Check exclusions
		for _, pattern := range args.Exclude {
			matched, _ := filepath.Match(pattern, d.Name())
			if matched {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		// Cross-mount check
		if args.OneFileSystem {
			if stat, ok := info.Sys().(*syscall.Stat_t); ok {
				if stat.Dev != rootDev {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
		}

		// Calculate blocks instead of apparent size
		var size int64
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			size = stat.Blocks * 512 // 512-byte blocks
		} else {
			size = info.Size() // fallback
		}

		if !info.IsDir() {
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

			if args.All {
				allFiles += fmt.Sprintf("%s\t%s\n", formatBytes(uint64(size)), path)
			}
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error during traversal: %v", err)
	}

	var result string
	
	if args.All && allFiles != "" {
		result += "Individual files:\n" + allFiles + "---\n"
	}

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
