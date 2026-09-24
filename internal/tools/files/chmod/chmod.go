// Package chmod implements files/chmod: change permission bits, never
// through a symbolic link.
package chmod

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon/internal/fsafe"
)

// Mode is a chmod mode that may arrive as a JSON string ("0755", "u+x") or
// as a JSON number - clients such as linuxctl send `0755` as the number
// 755. A number's decimal digits are the octal digits the caller typed, so
// 755 means mode 0755, never decimal 755 (= 01363).
type Mode string

func (m *Mode) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*m = Mode(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("mode must be a string or a number")
	}
	*m = Mode(n.String())
	return nil
}

// ChmodArgs defines the parameters for the files/chmod tool.
type ChmodArgs struct {
	Path       string `json:"path"`                 // Absolute path. Required.
	Mode       Mode   `json:"mode"`                 // Octal ("0644", "755", "4755") or symbolic ("u+x,go-w", "a=r"). Required.
	Recursive  bool   `json:"recursive,omitempty"`  // Apply to everything below a directory too (symlinks skipped).
	Privileged bool   `json:"privileged,omitempty"` // Run as root (if authorized in mcp-sudo.yaml).
}

func Chmod(argsJSON []byte) (string, error) {
	var args ChmodArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}
	if args.Path == "" || args.Mode == "" {
		return "", fmt.Errorf("path and mode are required")
	}
	apply, err := ParseMode(string(args.Mode))
	if err != nil {
		return "", err
	}

	root, err := fsafe.Open(args.Path)
	if err != nil {
		return "", err
	}
	defer root.Close()

	var log []string
	change := func(n *fsafe.Node) (bool, error) {
		old := n.ModeBits()
		mode := apply(old, n.IsDir())
		if mode == old {
			return false, nil
		}
		if err := n.Chmod(mode); err != nil {
			return false, fmt.Errorf("%s: %v", n.Path, err)
		}
		log = append(log, fmt.Sprintf("%s: %04o (%s) -> %04o (%s)", n.Path, old, render(old, n.IsDir()), mode, render(mode, n.IsDir())))
		return true, nil
	}

	if !args.Recursive {
		changed, err := change(root)
		if err != nil {
			return "", err
		}
		if !changed {
			return fmt.Sprintf("%s: mode already %04o (%s), unchanged\n", root.Path, root.ModeBits(), render(root.ModeBits(), root.IsDir())), nil
		}
		return log[0] + "\n", nil
	}
	res := fsafe.Walk(root, change)
	return summarize(log, res), nil
}

func summarize(log []string, res fsafe.WalkResult) string {
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
	return b.String() + "\n"
}

func render(mode uint32, dir bool) string {
	m := fs.FileMode(mode & 0o777)
	if mode&0o4000 != 0 {
		m |= fs.ModeSetuid
	}
	if mode&0o2000 != 0 {
		m |= fs.ModeSetgid
	}
	if mode&0o1000 != 0 {
		m |= fs.ModeSticky
	}
	s := []byte(m.Perm().String()) // "-rwxr-xr-x"
	if dir {
		s[0] = 'd'
	}
	for _, sp := range []struct {
		bit        fs.FileMode
		pos        int
		low, upper byte
	}{{fs.ModeSetuid, 3, 's', 'S'}, {fs.ModeSetgid, 6, 's', 'S'}, {fs.ModeSticky, 9, 't', 'T'}} {
		if m&sp.bit != 0 {
			if s[sp.pos] == 'x' {
				s[sp.pos] = sp.low
			} else {
				s[sp.pos] = sp.upper
			}
		}
	}
	return string(s)
}

// ParseMode parses an octal or symbolic chmod mode into a function that
// computes the new mode from the current one (and whether it's a
// directory, for the X permission).
func ParseMode(spec string) (func(old uint32, dir bool) uint32, error) {
	spec = strings.TrimSpace(spec)
	if spec != "" && strings.Trim(spec, "01234567") == "" {
		v, err := strconv.ParseUint(spec, 8, 32)
		if err != nil || len(spec) > 5 || v > 0o7777 {
			return nil, fmt.Errorf("invalid octal mode %q", spec)
		}
		// GNU chmod rule: on a directory, a numeric mode of up to four
		// digits never clears setuid/setgid (so "755" can't silently drop
		// the setgid bit that shared directories rely on); a five-digit
		// mode ("00755") sets the bits exactly.
		exact := len(spec) == 5
		return func(old uint32, dir bool) uint32 {
			if dir && !exact {
				return uint32(v) | old&0o6000
			}
			return uint32(v)
		}, nil
	}

	type clause struct {
		who   uint32 // mask of affected bits for u/g/o
		op    byte
		perms string
	}
	var clauses []clause
	for _, part := range strings.Split(spec, ",") {
		i := 0
		var who uint32
		for i < len(part) && strings.IndexByte("ugoa", part[i]) >= 0 {
			who |= map[byte]uint32{'u': 0o4700, 'g': 0o2070, 'o': 0o1007, 'a': 0o7777}[part[i]]
			i++
		}
		if who == 0 {
			who = 0o7777 // no who: all (umask is not applied here)
		}
		if i >= len(part) || strings.IndexByte("+-=", part[i]) < 0 {
			return nil, fmt.Errorf("invalid mode %q: expected octal (0755) or symbolic ([ugoa][+-=][rwxXst], e.g. u+x,go-w)", spec)
		}
		op := part[i]
		perms := part[i+1:]
		if strings.Trim(perms, "rwxXst") != "" {
			return nil, fmt.Errorf("invalid permissions %q in mode %q", perms, spec)
		}
		clauses = append(clauses, clause{who, op, perms})
	}
	return func(old uint32, dir bool) uint32 {
		mode := old
		for _, c := range clauses {
			var bits uint32
			for _, p := range c.perms {
				switch p {
				case 'r':
					bits |= 0o444
				case 'w':
					bits |= 0o222
				case 'x':
					bits |= 0o111
				case 'X': // execute only for dirs or if already executable by someone
					if dir || old&0o111 != 0 {
						bits |= 0o111
					}
				case 's':
					bits |= 0o6000
				case 't':
					bits |= 0o1000
				}
			}
			bits &= c.who
			switch c.op {
			case '+':
				mode |= bits
			case '-':
				mode &^= bits
			case '=':
				// Clear every bit this clause covers, then set the listed
				// ones. Like GNU chmod, a directory keeps its
				// setuid/setgid bits unless they're named explicitly.
				clear := c.who
				if dir {
					clear &^= 0o6000
				}
				mode = mode&^clear | bits
			}
		}
		return mode & 0o7777
	}, nil
}
