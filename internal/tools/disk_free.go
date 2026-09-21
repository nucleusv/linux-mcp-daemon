package tools

import (
	"encoding/json"
	"fmt"
	"syscall"
)

type GetDiskSpaceArgs struct {
	Path       string `json:"path"`
	Privileged bool   `json:"privileged,omitempty"`
}

// GetDiskSpace calculates filesystem usage statistics (equivalent to df -h).
func GetDiskSpace(rawArgs json.RawMessage) (string, error) {
	var args GetDiskSpaceArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
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

	// Calculate sizes in bytes
	totalBytes := stat.Blocks * uint64(stat.Bsize)
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes
	
	usePercent := float64(0)
	if totalBytes > 0 {
		usePercent = float64(usedBytes) / float64(totalBytes) * 100
	}

	// Format to human readable
	totalHR := formatBytes(totalBytes)
	usedHR := formatBytes(usedBytes)
	freeHR := formatBytes(freeBytes)

	result := fmt.Sprintf("Filesystem space on %s\n", args.Path)
	result += fmt.Sprintf("Total:     %s\n", totalHR)
	result += fmt.Sprintf("Used:      %s (%.1f%%)\n", usedHR, usePercent)
	result += fmt.Sprintf("Available: %s\n", freeHR)

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
