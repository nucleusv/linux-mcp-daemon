// Package manage implements cron/manage: read, list and replace crontabs.
//
// The master decides who may do what and whose crontab it is (see
// internal/rpc); this worker only carries out one already-authorized action,
// described by reserved "_" arguments the master sets and callers cannot:
//
//	_mode        "self"  the caller's own crontab, run as the caller, no root
//	             "other" another account's crontab: the worker is root and runs
//	                     `crontab` as a child with the target's credentials
//	             "list"  the users that have a crontab (root reads the spool)
//
// The crontab itself is read and written with the `crontab` command (a
// documented exception to "no CLI wrapping"): a user cannot write their own
// spool file, and the command checks the syntax, applies cron.allow and
// cron.deny, and tells cron.
package manage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
)

// MaxContent bounds a crontab written through mcpd.
const MaxContent = 64 << 10

// Args are the tool's arguments. The underscore fields are set by the master.
type Args struct {
	User         string  `json:"user,omitempty"`
	Content      *string `json:"content,omitempty"` // present = write, absent = read
	IfMatch      string  `json:"if_match,omitempty"`
	OutputFormat string  `json:"output_format,omitempty"`

	Mode         string   `json:"_mode,omitempty"`
	TargetUser   string   `json:"_target_user,omitempty"`
	TargetUID    uint32   `json:"_target_uid,omitempty"`
	TargetGID    uint32   `json:"_target_gid,omitempty"`
	TargetGroups []uint32 `json:"_target_groups,omitempty"`
	ViewUsers    []string `json:"_view_users,omitempty"`
	Info         bool     `json:"_info,omitempty"` // metadata only (the crontab://{user}/info template)
}

// Test hooks.
var (
	crontabBin = ""
	spoolDirs  = []string{"/var/spool/cron/crontabs", "/var/spool/cron"}
)

func Manage(argsJSON []byte) (string, error) {
	var a Args
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &a); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}
	switch a.Mode {
	case "list":
		return list(a)
	case "self", "other":
		if a.Mode == "other" && !config.ValidAccountName(a.TargetUser) {
			return "", fmt.Errorf("invalid account name %q", a.TargetUser)
		}
		if a.Content != nil {
			return write(a)
		}
		return read(a)
	}
	return "", errors.New("cron/manage was not prepared by the daemon (no mode)")
}

// crontabResult is what a read returns in json mode.
type crontabResult struct {
	User    string `json:"user"`
	Exists  bool   `json:"exists"`
	Lines   int    `json:"lines"`
	Jobs    int    `json:"jobs"`
	Bytes   int    `json:"bytes"`
	SHA256  string `json:"sha256"`
	Content string `json:"content,omitempty"`
}

func read(a Args) (string, error) {
	content, exists, err := current(a)
	if err != nil {
		return "", err
	}
	res := describe(a, content, exists)
	switch {
	case a.Info:
		res.Content = ""
		b, _ := json.MarshalIndent(res, "", "  ")
		return string(b), nil
	case a.OutputFormat == "json" || a.OutputFormat == "yaml" || a.OutputFormat == "table" || a.OutputFormat == "wide":
		b, _ := json.MarshalIndent(res, "", "  ")
		return string(b), nil
	}
	return content, nil // the raw crontab, exactly as `crontab -l` prints it
}

func describe(a Args, content string, exists bool) crontabResult {
	name := a.TargetUser
	if a.Mode == "self" || name == "" {
		name = a.User
		if name == "" {
			name = a.TargetUser
		}
	}
	return crontabResult{User: name, Exists: exists, Lines: countLines(content), Jobs: countJobs(content),
		Bytes: len(content), SHA256: sum(content), Content: content}
}

func write(a Args) (string, error) {
	content := *a.Content
	if len(content) > MaxContent {
		return "", fmt.Errorf("content is %d bytes, the limit is %d", len(content), MaxContent)
	}
	if strings.ContainsRune(content, 0) {
		return "", errors.New("content contains a NUL byte")
	}
	prev, exists, err := current(a)
	if err != nil {
		return "", err
	}
	if a.IfMatch != "" && !strings.EqualFold(a.IfMatch, sum(prev)) {
		return "", fmt.Errorf("the crontab changed since you read it (its sha256 is now %s, you passed %s) - read it again", sum(prev), a.IfMatch)
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n" // cron wants every line, the last one too, newline-terminated
	}
	switch {
	case content == "" && !exists:
		// nothing to clear
	case content == "":
		if _, stderr, code, err := run(a, nil, "-r"); err != nil || code != 0 {
			return "", crontabError("clearing the crontab", stderr, err)
		}
	default:
		if _, stderr, code, err := run(a, []byte(content), "-"); err != nil || code != 0 {
			return "", crontabError("crontab rejected the file - the old crontab is unchanged", stderr, err)
		}
	}
	res := describe(a, content, content != "")
	if a.OutputFormat == "json" || a.OutputFormat == "yaml" || a.OutputFormat == "table" || a.OutputFormat == "wide" {
		b, _ := json.MarshalIndent(map[string]interface{}{
			"user": res.User, "lines": res.Lines, "jobs": res.Jobs, "sha256": res.SHA256, "previous_sha256": sum(prev)}, "", "  ")
		return string(b), nil
	}
	return fmt.Sprintf("Crontab of %s replaced: %d lines, sha256 %s (was %s)", res.User, res.Lines, res.SHA256, sum(prev)), nil
}

// current reads the target's crontab; a missing one is empty, not an error.
func current(a Args) (string, bool, error) {
	out, stderr, code, err := run(a, nil, "-l")
	if err != nil {
		return "", false, crontabError("reading the crontab", stderr, err)
	}
	if code != 0 {
		if strings.Contains(stderr, "no crontab for") {
			return "", false, nil
		}
		return "", false, crontabError("reading the crontab", stderr, nil)
	}
	return out, true, nil
}

func crontabError(what, stderr string, err error) error {
	msg := strings.TrimSpace(stderr)
	if msg == "" && err != nil {
		msg = err.Error()
	}
	return fmt.Errorf("%s: %s", what, msg)
}

// run executes `crontab <args>` as the target: as the worker's own account
// for "self", as a child with the target's credentials for "other".
func run(a Args, stdin []byte, args ...string) (string, string, int, error) {
	bin, err := binary()
	if err != nil {
		return "", "", 0, err
	}
	cmd := exec.Command(bin, args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C"}
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	if a.Mode == "other" && a.TargetUID != 0 {
		if os.Geteuid() != 0 {
			return "", "", 0, errors.New("acting on another account's crontab needs the root worker")
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: a.TargetUID, Gid: a.TargetGID, Groups: a.TargetGroups}}
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return stdout.String(), stderr.String(), exitErr.ExitCode(), nil
	}
	return stdout.String(), stderr.String(), 0, err
}

func binary() (string, error) {
	if crontabBin != "" {
		return crontabBin, nil
	}
	for _, p := range []string{"/usr/bin/crontab", "/bin/crontab", "/usr/local/bin/crontab"} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", errors.New("crontab not found on this host - the cron package is not installed here (inside a container, the container's own filesystem is what a call sees)")
}

// list returns the users that have a crontab and that the master allowed the
// caller to view. Root reads the spool directory; the names it finds are
// treated as untrusted (checked, regular files only, no symlinks followed).
func list(a Args) (string, error) {
	dir := ""
	for _, d := range spoolDirs {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			dir = d
			break
		}
	}
	if dir == "" {
		return "", errors.New("no crontab spool directory on this host (cron is not installed?)")
	}
	allowed := map[string]bool{}
	for _, u := range a.ViewUsers {
		allowed[u] = true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading %s: %v", dir, err)
	}
	type row struct {
		User     string `json:"user"`
		Lines    int    `json:"lines"`
		Modified string `json:"modified"`
	}
	rows := []row{}
	for _, e := range entries {
		name := e.Name()
		if !allowed[name] || !config.ValidAccountName(name) {
			continue
		}
		path := filepath.Join(dir, name)
		st, err := os.Lstat(path)
		if err != nil || !st.Mode().IsRegular() {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		rows = append(rows, row{name, countLines(string(data)), st.ModTime().UTC().Format(time.RFC3339)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].User < rows[j].User })
	if a.OutputFormat == "json" || a.OutputFormat == "yaml" || a.OutputFormat == "table" || a.OutputFormat == "wide" {
		b, _ := json.MarshalIndent(rows, "", "  ")
		return string(b), nil
	}
	if len(rows) == 0 {
		return "No crontabs found for the accounts you may view.", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%-16s %-6s %s\n", "USER", "LINES", "MODIFIED")
	for _, r := range rows {
		fmt.Fprintf(&sb, "%-16s %-6d %s\n", r.User, r.Lines, r.Modified)
	}
	return sb.String(), nil
}

func sum(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(strings.TrimSuffix(s, "\n"), "\n") + 1
}

// countJobs counts the lines cron would run: not blank, not a comment, not an
// environment assignment (NAME=value).
func countJobs(s string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if i := strings.IndexByte(t, '='); i > 0 && !strings.ContainsAny(t[:i], " \t") {
			continue
		}
		n++
	}
	return n
}
