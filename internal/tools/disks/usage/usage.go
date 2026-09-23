package usage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

// GetUsageArgs defines the parameters for the get_disk_usage tool.
type GetUsageArgs struct {
	OutputFormat  string   `json:"output_format,omitempty"`   // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
	Path          string   `json:"path"`                      // Path is the absolute directory to start calculating from.
	MaxDepth      int      `json:"max_depth,omitempty"`       // MaxDepth determines how deep to recurse (0 for summarize only).
	OneFileSystem bool     `json:"one_file_system,omitempty"` // OneFileSystem prevents traversing into directories on different file systems.
	Exclude       []string `json:"exclude,omitempty"`         // Exclude contains file name patterns to ignore during traversal.
	All           bool     `json:"all,omitempty"`             // All includes individual file counts in the output, not just directories.
	ApparentSize  bool     `json:"apparent_size,omitempty"`   // ApparentSize forces the calculation of logical file sizes instead of physical blocks.
	Threshold     int64    `json:"threshold,omitempty"`       // Threshold filters files. Positive skips smaller files, negative skips larger files.
	SeparateDirs  bool     `json:"separate_dirs,omitempty"`   // SeparateDirs isolates a directory's size so it does not include subdirectories.
	Privileged    bool     `json:"privileged,omitempty"`      // Privileged executes the tool as the root user (if authorized in mcp-sudo.yaml).
}

// GetDiskUsage calculates disk usage by traversing a directory tree (equivalent to du -sh).
func Usage(argsJSON []byte) (string, error) {
	var args GetUsageArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
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
		rootDev = uint64(stat.Dev)
	}

	var rootDepth int
	if cleanPath == "/" || cleanPath == "." {
		rootDepth = 0
	} else {
		rootDepth = strings.Count(cleanPath, string(os.PathSeparator))
	}

	totalSize := int64(0)
	dirSizes := make(map[string]int64)
	var allFiles []entry
	// Hard-linked files are counted once, like du.
	seenInodes := make(map[[2]uint64]bool)

	err = filepath.WalkDir(cleanPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if path != cleanPath && excluded(args.Exclude, path, d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		stat, hasStat := info.Sys().(*syscall.Stat_t)

		// Cross-mount check
		if args.OneFileSystem && hasStat && uint64(stat.Dev) != rootDev {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if hasStat && !info.IsDir() && stat.Nlink > 1 {
			key := [2]uint64{uint64(stat.Dev), uint64(stat.Ino)}
			if seenInodes[key] {
				return nil
			}
			seenInodes[key] = true
		}

		// Calculate blocks instead of apparent size
		var size int64
		if !args.ApparentSize && hasStat {
			size = int64(stat.Blocks) * 512 // 512-byte blocks
		} else {
			size = info.Size()
		}

		// Everything - files and directories' own blocks, as du counts
		// them - adds to the total and to every ancestor directory at or
		// above max_depth (max_depth only controls which directories get a
		// printed line, not what gets summed into them). The threshold
		// filters printed entries only, never these sums - applying it per
		// file here used to drop every small file from the totals.
		totalSize += size
		if args.MaxDepth > 0 {
			dir := path
			if !info.IsDir() {
				dir = filepath.Dir(path)
			}
			for strings.HasPrefix(dir, cleanPath) {
				if strings.Count(dir, string(os.PathSeparator))-rootDepth <= args.MaxDepth || dir == cleanPath {
					dirSizes[dir] += size
				}
				if args.SeparateDirs || dir == cleanPath || dir == "/" || dir == "." {
					break // separate_dirs: only the immediate directory
				}
				dir = filepath.Dir(dir)
			}
		}

		if args.All && !info.IsDir() && passesThreshold(size, args.Threshold) {
			allFiles = append(allFiles, entry{path, size})
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error during traversal: %v", err)
	}

	// Largest first - the point of du is usually finding what's big.
	var dirs []entry
	for dir, size := range dirSizes {
		if passesThreshold(size, args.Threshold) {
			dirs = append(dirs, entry{dir, size})
		}
	}
	sortBySize(dirs)
	sortBySize(allFiles)

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		outObj := map[string]interface{}{
			"path":       cleanPath,
			"total_size": totalSize,
			"human_size": formatBytes(uint64(totalSize)),
		}
		if args.MaxDepth > 0 {
			sizes := make(map[string]int64, len(dirs))
			for _, e := range dirs {
				sizes[e.path] = e.size
			}
			outObj["directory_sizes"] = sizes
		}
		b, _ := json.Marshal(outObj)
		return string(b), nil
	}

	var result string

	if args.All && len(allFiles) > 0 {
		result += "Individual files (largest first):\n"
		for _, e := range allFiles {
			result += fmt.Sprintf("%s\t%s\n", formatBytes(uint64(e.size)), e.path)
		}
		result += "---\n"
	}

	if args.MaxDepth > 0 {
		result += fmt.Sprintf("Directory sizes (up to depth %d, largest first):\n", args.MaxDepth)
		for _, e := range dirs {
			result += fmt.Sprintf("%s\t%s\n", formatBytes(uint64(e.size)), e.path)
		}
		result += "---\n"
	}

	result += fmt.Sprintf("Total size of %s: %s\n", cleanPath, formatBytes(uint64(totalSize)))

	return result, nil
}

type entry struct {
	path string
	size int64
}

func sortBySize(entries []entry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].size != entries[j].size {
			return entries[i].size > entries[j].size
		}
		return entries[i].path < entries[j].path
	})
}

// passesThreshold mirrors du -t: a positive threshold hides entries smaller
// than it, a negative one hides entries larger than its absolute value.
func passesThreshold(size, threshold int64) bool {
	if threshold > 0 {
		return size >= threshold
	}
	if threshold < 0 {
		return size <= -threshold
	}
	return true
}

// excluded reports whether path matches any exclude pattern. Patterns
// containing a "/" match against the full path (so "/proc" or "/var/lib/*"
// work, and a matched directory excludes everything under it); others match
// the base name, like du --exclude.
func excluded(patterns []string, path, name string) bool {
	for _, pattern := range patterns {
		if strings.Contains(pattern, "/") {
			p := filepath.Clean(pattern)
			if path == p || strings.HasPrefix(path, p+"/") {
				return true
			}
			if matched, _ := filepath.Match(p, path); matched {
				return true
			}
			continue
		}
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
