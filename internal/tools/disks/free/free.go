package free

import (
	"encoding/json"
	"fmt"
	"syscall"
)

// GetFreeArgs defines the parameters for the get_disk_free tool.
type GetFreeArgs struct {
	OutputFormat string `json:\"output_format,omitempty\"` // OutputFormat specifies the desired output format (e.g. \"json\"). Defaults to text.
	Path          string `json:"path"`                     // Path is the absolute directory or mount point to check.
	Inodes        bool   `json:"inodes,omitempty"`         // Inodes requests the total and free inode index counts instead of byte usage.
	HumanReadable bool   `json:"human_readable,omitempty"` // HumanReadable formats the raw byte counts into human-readable strings (e.g. 24.5 GiB).
	Privileged    bool   `json:"privileged,omitempty"`     // Privileged executes the tool as the root user (if authorized in mcp-sudo.yaml).
}

// GetDiskFree calculates filesystem usage statistics (equivalent to df -h).
func Free(argsJSON []byte) (string, error) {
	var args GetFreeArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	var stat syscall.Statfs_t
	err := syscall.Statfs(args.Path, &stat)
	if err != nil {
		return "", fmt.Errorf("failed to get disk space for path %s: %v", args.Path, err)
	}

	if args.Inodes {
		totalInodes := stat.Files
		freeInodes := stat.Ffree
		usedInodes := totalInodes - freeInodes
		usePercent := float64(0)
		if totalInodes > 0 {
			usePercent = float64(usedInodes) / float64(totalInodes) * 100
		}
		result := fmt.Sprintf("Filesystem inodes on %s\n", args.Path)
		result += fmt.Sprintf("Total Inodes: %d\n", totalInodes)
		result += fmt.Sprintf("Used Inodes:  %d (%.1f%%)\n", usedInodes, usePercent)
		result += fmt.Sprintf("Free Inodes:  %d\n", freeInodes)
		return result, nil
	}

	// Calculate sizes in bytes
	totalBytes := stat.Blocks * uint64(stat.Bsize)
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes

	usePercent := float64(0)
	if totalBytes > 0 {
		usePercent = float64(usedBytes) / float64(totalBytes) * 100
	}

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		data := map[string]interface{}{
			"path":       args.Path,
			"total_bytes": totalBytes,
			"used_bytes":  usedBytes,
			"free_bytes":  freeBytes,
			"use_percent": usePercent,
		}
		if args.HumanReadable {
			data["total_human"] = formatBytes(totalBytes)
			data["used_human"] = formatBytes(usedBytes)
			data["free_human"] = formatBytes(freeBytes)
		}
		b, _ := json.Marshal(data)
		return string(b), nil
	}

	result := fmt.Sprintf("Filesystem space on %s\n", args.Path)
	if args.HumanReadable {
		result += fmt.Sprintf("Total:     %s\n", formatBytes(totalBytes))
		result += fmt.Sprintf("Used:      %s (%.1f%%)\n", formatBytes(usedBytes), usePercent)
		result += fmt.Sprintf("Available: %s\n", formatBytes(freeBytes))
	} else {
		result += fmt.Sprintf("Total:     %d\n", totalBytes)
		result += fmt.Sprintf("Used:      %d (%.1f%%)\n", usedBytes, usePercent)
		result += fmt.Sprintf("Available: %d\n", freeBytes)
	}

	return result, nil
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
