// Package chown implements files/chown: change owner and group, never
// through a symbolic link.
package chown

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon/internal/fsafe"
)

// ChownArgs defines the parameters for the files/chown tool.
type ChownArgs struct {
	Path       string `json:"path"`                 // Absolute path. Required.
	Owner      string `json:"owner"`                // "user", "user:group", ":group" or "user:" (user's login group); names or numeric ids. Required.
	Recursive  bool   `json:"recursive,omitempty"`  // Apply to everything below a directory too (symlinks skipped).
	Privileged bool   `json:"privileged,omitempty"` // Run as root - changing a file's owner requires it.
}

func Chown(argsJSON []byte) (string, error) {
	var args ChownArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}
	if args.Path == "" || args.Owner == "" {
		return "", fmt.Errorf("path and owner are required")
	}
	uid, gid, err := ParseOwner(args.Owner, "/etc/passwd", "/etc/group")
	if err != nil {
		return "", err
	}

	root, err := fsafe.Open(args.Path)
	if err != nil {
		return "", err
	}
	defer root.Close()

	users, groups := names("/etc/passwd"), names("/etc/group")
	who := func(u, g int) string { return nameOf(users, u) + ":" + nameOf(groups, g) }

	var log []string
	change := func(n *fsafe.Node) (bool, error) {
		oldU, oldG := n.Owner()
		newU, newG := oldU, oldG
		if uid >= 0 {
			newU = uid
		}
		if gid >= 0 {
			newG = gid
		}
		if newU == oldU && newG == oldG {
			return false, nil
		}
		if err := n.Chown(uid, gid); err != nil {
			return false, fmt.Errorf("%s: %v", n.Path, err)
		}
		log = append(log, fmt.Sprintf("%s: %s -> %s", n.Path, who(oldU, oldG), who(newU, newG)))
		return true, nil
	}

	if !args.Recursive {
		changed, err := change(root)
		if err != nil {
			return "", err
		}
		if !changed {
			u, g := root.Owner()
			return fmt.Sprintf("%s: owner already %s, unchanged\n", root.Path, who(u, g)), nil
		}
		return log[0] + "\n", nil
	}
	res := fsafe.Walk(root, change)
	var b strings.Builder
	for _, l := range log {
		b.WriteString(l + "\n")
	}
	fmt.Fprintf(&b, "changed %d, unchanged %d", res.Changed, res.Unchanged)
	if n := len(res.SkippedSymlinks); n > 0 {
		fmt.Fprintf(&b, ", skipped %d symlink(s) (never followed): %s", n, strings.Join(res.SkippedSymlinks, ", "))
	}
	if n := len(res.Errors); n > 0 {
		fmt.Fprintf(&b, "\n%d error(s):\n  %s", n, strings.Join(res.Errors, "\n  "))
	}
	return b.String() + "\n", nil
}

// ParseOwner turns "user", "user:group", ":group", "user:" or numeric
// forms into a uid and gid, -1 meaning "leave unchanged". "user:" means
// the user's login group, as in chown(1).
func ParseOwner(spec, passwdPath, groupPath string) (uid, gid int, err error) {
	userPart, groupPart, hasColon := strings.Cut(spec, ":")
	uid, gid = -1, -1
	if userPart != "" {
		uid, err = lookup(passwdPath, userPart, "user")
		if err != nil {
			return 0, 0, err
		}
	}
	switch {
	case groupPart != "":
		gid, err = lookup(groupPath, groupPart, "group")
		if err != nil {
			return 0, 0, err
		}
	case hasColon && userPart != "":
		gid, err = loginGroup(passwdPath, uid)
		if err != nil {
			return 0, 0, err
		}
	}
	if uid < 0 && gid < 0 {
		return 0, 0, fmt.Errorf("invalid owner %q: expected user, user:group, :group or user:", spec)
	}
	return uid, gid, nil
}

// lookup resolves a name or numeric id against a passwd/group file.
func lookup(path, name, kind string) (int, error) {
	if id, err := strconv.Atoi(name); err == nil && id >= 0 {
		return id, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Split(line, ":")
		if len(f) >= 3 && f[0] == name {
			return strconv.Atoi(f[2])
		}
	}
	return 0, fmt.Errorf("no such %s: %q", kind, name)
}

func loginGroup(passwdPath string, uid int) (int, error) {
	data, err := os.ReadFile(passwdPath)
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Split(line, ":")
		if len(f) >= 4 && f[2] == strconv.Itoa(uid) {
			return strconv.Atoi(f[3])
		}
	}
	return 0, fmt.Errorf("no login group for uid %d", uid)
}

func names(path string) map[int]string {
	m := map[int]string{}
	data, _ := os.ReadFile(path)
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Split(line, ":")
		if len(f) >= 3 {
			if id, err := strconv.Atoi(f[2]); err == nil {
				if _, dup := m[id]; !dup {
					m[id] = f[0]
				}
			}
		}
	}
	return m
}

func nameOf(m map[int]string, id int) string {
	if n, ok := m[id]; ok {
		return n
	}
	return strconv.Itoa(id)
}
