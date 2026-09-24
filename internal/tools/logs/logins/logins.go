package logins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// GetLoginsArgs are the tool's input arguments.
type GetLoginsArgs struct {
	Type       string `json:"type,omitempty"`       // "success" (default, wraps `last`) or "failed" (wraps `lastb`).
	Limit      int    `json:"limit,omitempty"`      // Only return this many most recent entries.
	User       string `json:"user,omitempty"`       // Only return entries for this username.
	Privileged bool   `json:"privileged,omitempty"` // Run as root - typically required for type: "failed" (btmp is usually root-only readable).
}

// validUsername is deliberately strict: last/lastb take the username as a
// positional argument, not a shell string, so there's no injection risk in
// the traditional sense - but a value starting with "-" could still be
// misread as a flag by the wrapped binary, so this guards against that
// class of "unintended behavior via crafted input" rather than assuming
// exec.Command's lack of a shell makes any input automatically safe.
var validUsername = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]*$`)

// List returns login records by wrapping `last`/`lastb`. These are not
// natively reimplemented: wtmp/btmp are fixed-size binary C structs whose
// exact layout is glibc/arch-specific, unlike the plain-text formats this
// daemon otherwise hand-parses (dpkg's status file, /proc/limits) - getting
// the layout wrong would silently produce garbage instead of an error, so
// this follows the project's documented precedent for wrapping a CLI
// (smartctl, traceroute, file) instead of hand-rolling something fragile.
//
// Returns raw text, not JSON, regardless of output_format: last/lastb have
// no reliable structured output mode, and their text format (variable-width
// host field, "still logged in"/"gone - no logout" special cases) isn't
// safe to hand-parse into JSON without risking confidently-wrong structured
// data - honest raw text beats a fragile custom parser here.
func List(argsJSON []byte) (string, error) {
	var args GetLoginsArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	bin := "last"
	switch args.Type {
	case "", "success":
		bin = "last"
	case "failed":
		bin = "lastb"
	default:
		return "", fmt.Errorf("invalid type %q: must be \"success\" or \"failed\"", args.Type)
	}

	var cmdArgs []string
	if args.Limit > 0 {
		cmdArgs = append(cmdArgs, "-n", fmt.Sprintf("%d", args.Limit))
	}
	if args.User != "" {
		if !validUsername.MatchString(args.User) {
			return "", fmt.Errorf("invalid user %q", args.User)
		}
		cmdArgs = append(cmdArgs, args.User)
	}

	cmd := exec.Command(bin, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%s failed: %v: %s", bin, err, strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("%s failed: %v", bin, err)
	}

	// Drop the trailing "wtmp begins ..."/"btmp begins ..." summary line -
	// it's metadata about the log file, not a login record.
	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if n := len(lines); n > 0 && (strings.HasPrefix(lines[n-1], "wtmp begins") || strings.HasPrefix(lines[n-1], "btmp begins")) {
		lines = lines[:n-1]
	}

	return strings.Join(lines, "\n") + "\n", nil
}
