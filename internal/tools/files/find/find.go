package findfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon/internal/fsafe"
)

// FindFileArgs defines the parameters for the files/find tool.
type FindFileArgs struct {
	Path         string `json:"path"`                    // Path is the starting directory for the search. Defaults to "/".
	Name         string `json:"name,omitempty"`          // Name is the glob pattern to match filenames.
	Type         string `json:"type,omitempty"`          // Type filters by file type ('f' for file, 'd' for directory, 'l' for symlink).
	Mtime        string `json:"mtime,omitempty"`         // Mtime filters by modification time (e.g. "+7" for older than 7 days).
	Size         string `json:"size,omitempty"`          // Size filters by file size (e.g. "+100M" for larger than 100MB).
	MaxDepth     int    `json:"max_depth,omitempty"`     // MaxDepth restricts the depth of the search.
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format. Defaults to text.
	// NoFollow is set by the daemon (never the caller) when this runs as
	// root under a paths: restriction: a symlink in any component of path
	// is then refused instead of followed out of the allowed directories.
	NoFollow bool `json:"_no_follow,omitempty"`
}

// Match is one search result.
type Match struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Type string `json:"type"` // find's %y: f, d, l, ...
}

// virtualFS are pseudo-filesystems pruned from searches that would
// otherwise descend into them: they hold no real files, are huge (/proc
// alone lists every process's fd table), and report nonsense sizes
// (/proc/kcore claims to be as large as the address space).
var virtualFS = []string{"/proc", "/sys", "/dev", "/run"}

var (
	validType  = regexp.MustCompile(`^[bcdpflsD](,[bcdpflsD])*$`)
	validMtime = regexp.MustCompile(`^[+-]?[0-9]+$`)
	validSize  = regexp.MustCompile(`^[+-]?[0-9]+[bcwkMG]?$`)
)

func Find(argsJSON []byte) (string, error) {
	var args FindFileArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	cmdArgs, err := buildArgs(args)
	if err != nil {
		return "", err
	}

	// find never follows symlinks while walking (-P), but the kernel does
	// resolve every component of the start path. With NoFollow, the start
	// directory is opened without following any, made the working
	// directory, and searched as "." - results are mapped back below.
	root := ""
	if args.NoFollow {
		root = cmdArgs[0]
		n, err := fsafe.Open(root)
		if err != nil {
			return "", fmt.Errorf("cannot search %s: %v", root, err)
		}
		err = n.Chdir()
		n.Close()
		if err != nil {
			return "", fmt.Errorf("cannot search %s: %v", root, err)
		}
		for i, a := range cmdArgs {
			switch {
			case a == root:
				cmdArgs[i] = "."
			case strings.HasPrefix(a, root+"/"):
				cmdArgs[i] = "." + strings.TrimPrefix(a, root)
			}
		}
	}

	cmd := exec.Command("find", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// find often exits with 1 if permission is denied on some directories,
		// but still returns valid output for others.
		if len(stdout.Bytes()) == 0 {
			return "", fmt.Errorf("find error: %v, stderr: %s", err, stderr.String())
		}
	}

	matches := parseOutput(stdout.String())
	if root != "" {
		for i := range matches {
			if matches[i].Path == "." {
				matches[i].Path = root
			} else {
				matches[i].Path = filepath.Join(root, strings.TrimPrefix(matches[i].Path, "./"))
			}
		}
	}

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		b, err := json.Marshal(matches)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %v", err)
		}
		return string(b), nil
	}

	var sb strings.Builder
	for _, m := range matches {
		size := ""
		if m.Type != "d" {
			size = formatBytes(m.Size)
		}
		sb.WriteString(fmt.Sprintf("%-10s %s\n", size, m.Path))
	}
	return sb.String(), nil
}

// buildArgs validates every user-supplied value and builds find's argv.
// Validation matters here: find reads any argument starting with "-" as an
// expression, so an unchecked path like "-delete" would become
// `find -delete ...` - deleting everything under the working directory,
// as root for privileged calls.
func buildArgs(args FindFileArgs) ([]string, error) {
	searchPath := args.Path
	if searchPath == "" {
		searchPath = "/"
	}
	if !strings.HasPrefix(searchPath, "/") {
		return nil, fmt.Errorf("path must be absolute, got %q", args.Path)
	}
	searchPath = filepath.Clean(searchPath)
	if args.Type != "" && !validType.MatchString(args.Type) {
		return nil, fmt.Errorf("invalid type %q (use f, d, l, b, c, p, s)", args.Type)
	}
	if args.Mtime != "" && !validMtime.MatchString(args.Mtime) {
		return nil, fmt.Errorf("invalid mtime %q (e.g. 7, +7, -1)", args.Mtime)
	}
	if args.Size != "" && !validSize.MatchString(args.Size) {
		return nil, fmt.Errorf("invalid size %q (e.g. +100M, -10k)", args.Size)
	}

	cmdArgs := []string{searchPath}
	if args.MaxDepth > 0 {
		cmdArgs = append(cmdArgs, "-maxdepth", strconv.Itoa(args.MaxDepth))
	}

	// Prune virtual filesystems below the search root - but not when the
	// caller is explicitly searching inside one (path: "/proc/1").
	var prune []string
	for _, v := range virtualFS {
		if (searchPath == "/" || strings.HasPrefix(v, searchPath+"/")) && v != searchPath {
			prune = append(prune, v)
		}
	}
	if len(prune) > 0 {
		cmdArgs = append(cmdArgs, "(")
		for i, p := range prune {
			if i > 0 {
				cmdArgs = append(cmdArgs, "-o")
			}
			cmdArgs = append(cmdArgs, "-path", p)
		}
		cmdArgs = append(cmdArgs, ")", "-prune", "-o")
	}

	if args.Name != "" {
		cmdArgs = append(cmdArgs, "-name", args.Name) // an operand of -name, never parsed as an expression
	}
	if args.Type != "" {
		cmdArgs = append(cmdArgs, "-type", args.Type)
	}
	if args.Mtime != "" {
		cmdArgs = append(cmdArgs, "-mtime", args.Mtime)
	}
	if args.Size != "" {
		cmdArgs = append(cmdArgs, "-size", args.Size)
	}
	// Explicit action, so pruned directories themselves aren't printed.
	// NUL-free, tab-separated: size, type, path (paths may contain spaces).
	cmdArgs = append(cmdArgs, "-printf", `%s\t%y\t%p\n`)
	return cmdArgs, nil
}

func parseOutput(out string) []Match {
	matches := []Match{}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		size, _ := strconv.ParseInt(parts[0], 10, 64)
		matches = append(matches, Match{Path: parts[2], Size: size, Type: parts[1]})
	}
	return matches
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
