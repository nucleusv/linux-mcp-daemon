package read

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ProcessReadArgs defines the parameters for the process://{pid}/* resource.
type ProcessReadArgs struct {
	PID    int    `json:"pid"`
	Target string `json:"target"` // "status", "cmdline", "environ", "limits", "open_files"
}

func Read(argsJSON []byte) (string, error) {
	var args ProcessReadArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if args.PID <= 0 {
		return "", fmt.Errorf("invalid pid: %d", args.PID)
	}

	switch args.Target {
	case "status":
		content, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", args.PID))
		if err != nil {
			return "", fmt.Errorf("failed to read status: %v", err)
		}
		return string(content), nil

	case "cmdline":
		content, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", args.PID))
		if err != nil {
			return "", fmt.Errorf("failed to read cmdline: %v", err)
		}
		// cmdline is null-byte separated
		argsList := strings.Split(string(content), "\x00")
		// the last element might be empty due to trailing null byte
		if len(argsList) > 0 && argsList[len(argsList)-1] == "" {
			argsList = argsList[:len(argsList)-1]
		}
		out, _ := json.MarshalIndent(argsList, "", "  ")
		return string(out), nil

	case "environ":
		content, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", args.PID))
		if err != nil {
			return "", fmt.Errorf("failed to read environ: %v", err)
		}
		// environ is null-byte separated KEY=VALUE pairs
		envList := strings.Split(string(content), "\x00")
		envMap := make(map[string]string)
		for _, env := range envList {
			if env == "" {
				continue
			}
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}
		out, _ := json.MarshalIndent(envMap, "", "  ")
		return string(out), nil

	case "limits":
		content, err := os.ReadFile(fmt.Sprintf("/proc/%d/limits", args.PID))
		if err != nil {
			return "", fmt.Errorf("failed to read limits: %v", err)
		}
		// /proc/{pid}/limits is a fixed-width table, e.g.:
		//   Limit                     Soft Limit           Hard Limit           Units
		//   Max open files            1024                 524288               files
		// Limit names can themselves contain single spaces ("Max open files"),
		// so a naive whitespace split misaligns columns - splitting on runs of
		// 2+ spaces (the column padding) instead keeps each name intact.
		splitRe := regexp.MustCompile(`\s{2,}`)
		var limits []map[string]string
		for i, line := range strings.Split(string(content), "\n") {
			if i == 0 || strings.TrimSpace(line) == "" {
				continue // header row, or trailing blank line
			}
			fields := splitRe.Split(strings.TrimRight(line, " \t"), -1)
			entry := map[string]string{"name": strings.TrimSpace(fields[0])}
			if len(fields) > 1 {
				entry["soft"] = strings.TrimSpace(fields[1])
			}
			if len(fields) > 2 {
				entry["hard"] = strings.TrimSpace(fields[2])
			}
			if len(fields) > 3 {
				entry["units"] = strings.TrimSpace(fields[3])
			}
			limits = append(limits, entry)
		}
		out, _ := json.MarshalIndent(limits, "", "  ")
		return string(out), nil

	case "open_files":
		dirPath := fmt.Sprintf("/proc/%d/fd", args.PID)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return "", fmt.Errorf("failed to read open files: %v", err)
		}
		type openFile struct {
			FD     string `json:"fd"`
			Target string `json:"target"`
		}
		files := make([]openFile, 0, len(entries))
		for _, e := range entries {
			// Each entry is a symlink; its target is a real file path for a
			// plain file, or a pseudo-path like "socket:[12345]" or
			// "pipe:[12345]" for non-file descriptors.
			target, err := os.Readlink(filepath.Join(dirPath, e.Name()))
			if err != nil {
				target = fmt.Sprintf("<unreadable: %v>", err)
			}
			files = append(files, openFile{FD: e.Name(), Target: target})
		}
		sort.Slice(files, func(i, j int) bool {
			ni, _ := strconv.Atoi(files[i].FD)
			nj, _ := strconv.Atoi(files[j].FD)
			return ni < nj
		})
		out, _ := json.MarshalIndent(files, "", "  ")
		return string(out), nil

	default:
		return "", fmt.Errorf("unsupported target: %s", args.Target)
	}
}

// CleanUp cleans up string paths
func CleanUp(s string) string {
	return filepath.Clean(s)
}
