package mounts

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// GetMountsArgs are the tool's input arguments.
type GetMountsArgs struct {
	OutputFormat string `json:"output_format,omitempty"` // Desired output format (json/yaml/table/wide). Defaults to text.
	FSType       string `json:"fs_type,omitempty"`        // Optional filter: only include mounts of this filesystem type (e.g. "ext4", "overlay").
	Privileged   bool   `json:"privileged,omitempty"`     // Run as root.
}

// Mount describes a single mounted filesystem.
type Mount struct {
	Device     string `json:"device"`
	MountPoint string `json:"mount_point"`
	FSType     string `json:"fs_type"`
	Options    string `json:"options"`
}

// mtabEscape matches the octal escape sequences /proc/self/mounts uses for
// whitespace and backslashes in device/mountpoint paths (the same
// convention as /etc/fstab and /etc/mtab).
var mtabEscape = regexp.MustCompile(`\\[0-7]{3}`)

func unescapeMtab(s string) string {
	return mtabEscape.ReplaceAllStringFunc(s, func(m string) string {
		n, err := strconv.ParseInt(m[1:], 8, 32)
		if err != nil {
			return m
		}
		return string(rune(n))
	})
}

// List returns the filesystems currently mounted, natively parsing
// /proc/self/mounts (equivalent to `mount`/`findmnt`'s basic view). Combine
// with "privileged": true on a daemon deployed with worker.containerized:
// true to see the real host's mount table instead of this container's own.
func List(argsJSON []byte) (string, error) {
	var args GetMountsArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	// /proc/thread-self, not /proc/self, as a precaution: after
	// JoinHostMountNamespace does a per-thread setns+chroot (main.go's
	// worker-mode branch, when privileged + worker.containerized), "self"
	// is documented as potentially resolving via the thread-group *leader*
	// task rather than the calling OS thread for namespace-sensitive files.
	// /proc/thread-self (Linux 3.17+) always refers to the calling thread
	// unambiguously. Not a confirmed bug in practice here (see
	// ARCHITECTURE.md) - just removing a theoretical risk at no cost.
	content, err := os.ReadFile("/proc/thread-self/mounts")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/thread-self/mounts: %v", err)
	}

	var result []Mount
	for _, line := range strings.Split(string(content), "\n") {
		if line == "" {
			continue
		}
		// Format: device mountpoint fstype options dump-freq pass-number
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		m := Mount{
			Device:     unescapeMtab(fields[0]),
			MountPoint: unescapeMtab(fields[1]),
			FSType:     fields[2],
			Options:    fields[3],
		}
		if args.FSType != "" && m.FSType != args.FSType {
			continue
		}
		result = append(result, m)
	}

	sort.Slice(result, func(i, j int) bool { return result[i].MountPoint < result[j].MountPoint })

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		b, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %v", err)
		}
		return string(b), nil
	}

	var sb strings.Builder
	for _, m := range result {
		sb.WriteString(fmt.Sprintf("%s on %s type %s (%s)\n", m.Device, m.MountPoint, m.FSType, m.Options))
	}
	return sb.String(), nil
}
