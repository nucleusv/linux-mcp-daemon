package partitions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PartitionsArgs defines the parameters for the disks/partitions tool.
type PartitionsArgs struct {
	Device       string `json:"device,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

// Partition describes one partition, read natively from /sys/class/block.
type Partition struct {
	Device      string `json:"device"`
	ParentDisk  string `json:"parent_disk"`
	Number      int    `json:"number"`
	StartSector uint64 `json:"start_sector"`
	SizeSectors uint64 `json:"size_sectors"`
	SizeBytes   uint64 `json:"size_bytes"`
}

func readUint(path string) (uint64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func Partitions(argsJSON []byte) (string, error) {
	var args PartitionsArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	entries, err := os.ReadDir("/sys/class/block")
	if err != nil {
		return "", fmt.Errorf("failed to read /sys/class/block: %v", err)
	}

	var out []Partition
	for _, entry := range entries {
		name := entry.Name()

		// Whole disks have no "partition" file; only partitions do. This is
		// the kernel's own way of telling the two apart, since /sys/class/block
		// lists both flatly with no other structural distinction.
		number, isPartition := readUint(filepath.Join("/sys/class/block", name, "partition"))
		if !isPartition {
			continue
		}

		// /sys/class/block/<name> is a symlink into the parent disk's real
		// sysfs directory (e.g. .../block/vda/vda1) - resolving it is how we
		// recover which disk this partition belongs to, without relying on
		// brittle name-prefix guessing (nvme0n1p1, mmcblk0p1, sda1, ... all
		// differ in how the partition number is appended to the disk name).
		real, err := filepath.EvalSymlinks(filepath.Join("/sys/class/block", name))
		if err != nil {
			continue
		}
		parent := filepath.Base(filepath.Dir(real))

		if args.Device != "" && parent != args.Device {
			continue
		}

		start, _ := readUint(filepath.Join("/sys/class/block", name, "start"))
		size, _ := readUint(filepath.Join("/sys/class/block", name, "size"))

		out = append(out, Partition{
			Device:      name,
			ParentDisk:  parent,
			Number:      int(number),
			StartSector: start,
			SizeSectors: size,
			SizeBytes:   size * 512,
		})
	}

	if len(out) == 0 && args.Device != "" {
		return "", fmt.Errorf("no partitions found for device %q (check disks/list for valid device names)", args.Device)
	}

	if args.OutputFormat == "" {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%-14s %-10s %-6s %-14s %-14s %-14s\n", "DEVICE", "PARENT", "NUM", "START(SECT)", "SIZE(SECT)", "SIZE(BYTES)"))
		for _, p := range out {
			sb.WriteString(fmt.Sprintf("%-14s %-10s %-6d %-14d %-14d %-14d\n", p.Device, p.ParentDisk, p.Number, p.StartSector, p.SizeSectors, p.SizeBytes))
		}
		return sb.String(), nil
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode response: %v", err)
	}
	return string(b), nil
}
