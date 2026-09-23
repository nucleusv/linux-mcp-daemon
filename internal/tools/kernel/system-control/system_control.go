package system_control

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SystemControlArgs defines the parameters for the kernel/system-control tool.
type SystemControlArgs struct {
	Key        string `json:"key"`                  // Key is the kernel parameter to read or write (e.g. "net.ipv4.ip_forward"). Required unless ReadAll is true.
	Value      string `json:"value,omitempty"`      // Value is the value to write to the parameter. If provided, writes the value.
	ReadAll    bool   `json:"read_all,omitempty"`   // ReadAll reads all kernel parameters (sysctl -a).
	Privileged bool   `json:"privileged,omitempty"` // Privileged runs as root - required for writes.
}

// procSys is the sysctl tree. A variable only so tests can point it at a
// temporary directory.
var procSys = "/proc/sys"

// validKey matches sysctl parameter names (dotted or slash-separated) and
// can never start with "-" or contain "..".
var validKey = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.\-/]*$`)

// SystemControl reads and writes kernel parameters natively through
// /proc/sys - the same files the sysctl binary uses - instead of running
// sysctl, so there is no command line to inject into. Output matches
// sysctl's "key = value" format.
func SystemControl(argsJSON []byte) (string, error) {
	var args SystemControlArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	if args.Key == "" {
		if !args.ReadAll {
			return "", fmt.Errorf("key argument is required unless read_all is true")
		}
		return readTree(procSys)
	}

	path, err := KeyPath(args.Key)
	if err != nil {
		return "", err
	}

	if args.Value != "" {
		if strings.ContainsAny(args.Value, "\n\x00") {
			return "", fmt.Errorf("invalid value: must be a single line")
		}
		// O_WRONLY without O_CREATE/O_TRUNC: procfs entries can't be
		// created, and a missing key must fail rather than be invented.
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return "", fmt.Errorf("cannot write %s: %v", NormalizeKey(args.Key), err)
		}
		_, werr := f.WriteString(args.Value)
		cerr := f.Close()
		if werr != nil {
			return "", fmt.Errorf("cannot write %s: %v", NormalizeKey(args.Key), werr)
		}
		if cerr != nil {
			return "", fmt.Errorf("cannot write %s: %v", NormalizeKey(args.Key), cerr)
		}
		// Like sysctl -w, report the value as the kernel now holds it.
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("cannot stat %s: %v", NormalizeKey(args.Key), err)
	}
	if info.IsDir() {
		// "sysctl net.ipv4" prints the whole subtree.
		return readTree(path)
	}
	value, err := readValue(path)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %v", NormalizeKey(args.Key), err)
	}
	return fmt.Sprintf("%s = %s\n", NormalizeKey(args.Key), value), nil
}

// NormalizeKey returns the dotted form of a key: "net/ipv4/ip_forward" and
// "net.ipv4.ip_forward" both become "net.ipv4.ip_forward".
func NormalizeKey(key string) string {
	return strings.Trim(strings.ReplaceAll(key, "/", "."), ".")
}

// KeyPath maps a sysctl key to its file under /proc/sys, rejecting
// anything that would resolve outside that tree.
func KeyPath(key string) (string, error) {
	if !validKey.MatchString(key) || strings.Contains(key, "..") {
		return "", fmt.Errorf("invalid key %q: expected a sysctl name like net.ipv4.ip_forward", key)
	}
	rel := key
	if !strings.Contains(key, "/") {
		// Dotted form. Like sysctl, a dotted key can't address a
		// component that itself contains a dot (e.g. VLAN interface
		// eth0.100) - use the slash form for those.
		rel = strings.ReplaceAll(key, ".", "/")
	}
	path := filepath.Join(procSys, filepath.Clean("/"+rel))
	if path != procSys && !strings.HasPrefix(path, procSys+"/") {
		return "", fmt.Errorf("invalid key %q", key)
	}
	return path, nil
}

func readValue(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(b), "\n"), nil
}

// readTree prints every readable parameter under dir, sorted, like
// sysctl -a. Entries that can't be read (write-only triggers such as
// vm.compact_memory, or root-only ones) are skipped, as sysctl -a does.
func readTree(dir string) (string, error) {
	var lines []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		value, err := readValue(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(procSys, path)
		key := strings.ReplaceAll(rel, string(filepath.Separator), ".")
		// Multi-line values (e.g. dev.cdrom.info) print one line each,
		// as sysctl does.
		for _, v := range strings.Split(value, "\n") {
			lines = append(lines, fmt.Sprintf("%s = %s", key, v))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(lines)
	if len(lines) == 0 {
		return "", nil
	}
	return strings.Join(lines, "\n") + "\n", nil
}
