package listfiles

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/fsafe"
	"golang.org/x/sys/unix"
)

// GetListOfFilesArgs defines the parameters for the files/list tool.
type GetListOfFilesArgs struct {
	OutputFormat  string `json:"output_format,omitempty"`  // json/yaml/table/wide for structured output; text otherwise.
	Path          string `json:"path"`                     // Directory to list. Required.
	All           bool   `json:"all,omitempty"`            // Include dotfiles, "." and ".." (ls -a).
	Long          *bool  `json:"long,omitempty"`           // Long listing (ls -l). Default true; false lists names only.
	HumanReadable bool   `json:"human_readable,omitempty"` // Sizes like 4.0K, 1.2M (ls -h).
	Sort          string `json:"sort,omitempty"`           // name (default), size (largest first), time (newest first).
	Reverse       bool   `json:"reverse,omitempty"`        // Reverse the sort order (ls -r).
	DirsFirst     bool   `json:"dirs_first,omitempty"`     // Directories before files (ls --group-directories-first).
	NumericIDs    bool   `json:"numeric_ids,omitempty"`    // Show uid/gid numbers instead of names (ls -n).
	Privileged    bool   `json:"privileged,omitempty"`     // Run as root (if authorized in mcp-sudo.yaml).
	// NoFollow is set by the daemon (never the caller) when this runs as
	// root under a paths: restriction: a symlink in any path component is
	// then refused instead of followed out of the allowed directories.
	NoFollow bool `json:"_no_follow,omitempty"`
}

// Entry is one directory entry, with everything `ls -l` shows.
type Entry struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // file, dir, symlink, char, block, fifo, socket
	Mode      string `json:"mode"` // ls-style, e.g. -rw-r--r--, drwxrwxrwt
	ModeOctal string `json:"mode_octal"`
	Links     uint64 `json:"links"`
	Owner     string `json:"owner"`
	Group     string `json:"group"`
	UID       uint32 `json:"uid"`
	GID       uint32 `json:"gid"`
	Size      int64  `json:"size"`
	Device    string `json:"device,omitempty"` // "major, minor" for char/block devices
	Modified  string `json:"modified"`         // 2006-01-02 15:04:05, as before
	IsDir     bool   `json:"is_dir"`
	Target    string `json:"target,omitempty"` // symlink target

	modTime time.Time
	blocks  int64
}

// ListOfFiles lists a directory like `ls -la`. Entries are read with
// lstat: a symlink is reported as a symlink (with its target), never
// followed.
func ListOfFiles(argsJSON []byte) (string, error) {
	var args GetListOfFilesArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("path argument is required")
	}
	switch args.Sort {
	case "", "name", "size", "time":
	default:
		return "", fmt.Errorf("invalid sort %q (name, size, time)", args.Sort)
	}
	long := args.Long == nil || *args.Long

	dir := filepath.Clean(args.Path)
	parent := filepath.Dir(dir)
	if args.NoFollow {
		// Read the directory through a descriptor reached without symlinks;
		// entries are then looked up relative to it (names are shown as is).
		n, err := fsafe.Open(dir)
		if err != nil {
			return "", fmt.Errorf("failed to read directory: %v", err)
		}
		defer n.Close()
		if !n.IsDir() {
			return "", fmt.Errorf("failed to read directory: %s: not a directory", dir)
		}
		dir = n.ProcPath()
		parent = dir + "/.."
	}
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %v", err)
	}

	users, groups := idNames("/etc/passwd"), idNames("/etc/group")
	var entries []Entry
	add := func(name, path string) {
		info, err := os.Lstat(path)
		if err != nil {
			entries = append(entries, Entry{Name: name, Type: "?", Mode: "??????????"})
			return
		}
		entries = append(entries, makeEntry(name, path, info, users, groups))
	}
	if args.All {
		add(".", dir)
		add("..", parent)
	}
	for _, e := range dirEntries {
		if !args.All && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		add(e.Name(), filepath.Join(dir, e.Name()))
	}
	sortEntries(entries, args.Sort, args.Reverse, args.DirsFirst)

	switch args.OutputFormat {
	case "json", "yaml", "table", "wide":
		if entries == nil {
			entries = []Entry{}
		}
		b, err := json.Marshal(entries)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	if len(entries) == 0 {
		return "Directory is empty.\n", nil
	}
	if !long {
		var b strings.Builder
		for _, e := range entries {
			b.WriteString(e.Name)
			if e.IsDir && e.Name != "." && e.Name != ".." {
				b.WriteString("/")
			}
			b.WriteString("\n")
		}
		return b.String(), nil
	}
	return formatLong(entries, args.HumanReadable, args.NumericIDs), nil
}

func makeEntry(name, path string, info fs.FileInfo, users, groups map[uint32]string) Entry {
	e := Entry{
		Name:      name,
		Type:      fileType(info.Mode()),
		Mode:      modeString(info.Mode()),
		ModeOctal: fmt.Sprintf("%04o", unixPerm(info.Mode())),
		Size:      info.Size(),
		Modified:  info.ModTime().Format("2006-01-02 15:04:05"),
		IsDir:     info.IsDir(),
		modTime:   info.ModTime(),
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		e.Links = uint64(st.Nlink)
		e.UID, e.GID = st.Uid, st.Gid
		e.blocks = int64(st.Blocks)
		if e.Type == "char" || e.Type == "block" {
			rdev := uint64(st.Rdev)
			e.Device = fmt.Sprintf("%d, %d", unix.Major(rdev), unix.Minor(rdev))
		}
	}
	e.Owner = nameOr(users, e.UID)
	e.Group = nameOr(groups, e.GID)
	if info.Mode()&fs.ModeSymlink != 0 {
		e.Target, _ = os.Readlink(path)
	}
	return e
}

func fileType(m fs.FileMode) string {
	switch {
	case m&fs.ModeSymlink != 0:
		return "symlink"
	case m.IsDir():
		return "dir"
	case m&fs.ModeCharDevice != 0:
		return "char"
	case m&fs.ModeDevice != 0:
		return "block"
	case m&fs.ModeNamedPipe != 0:
		return "fifo"
	case m&fs.ModeSocket != 0:
		return "socket"
	}
	return "file"
}

// unixPerm converts Go's FileMode to the classic 12-bit Unix mode
// (setuid/setgid/sticky + rwx).
func unixPerm(m fs.FileMode) uint32 {
	p := uint32(m.Perm())
	if m&fs.ModeSetuid != 0 {
		p |= 04000
	}
	if m&fs.ModeSetgid != 0 {
		p |= 02000
	}
	if m&fs.ModeSticky != 0 {
		p |= 01000
	}
	return p
}

// modeString renders a mode the way ls does - Go's FileMode.String()
// differs ("Lrwxrwxrwx" for symlinks, "urwx..." for setuid).
func modeString(m fs.FileMode) string {
	typeChar := map[string]byte{"file": '-', "dir": 'd', "symlink": 'l', "char": 'c', "block": 'b', "fifo": 'p', "socket": 's'}[fileType(m)]
	b := []byte{typeChar, '-', '-', '-', '-', '-', '-', '-', '-', '-'}
	perm := m.Perm()
	for i, c := range "rwxrwxrwx" {
		if perm&(1<<uint(8-i)) != 0 {
			b[i+1] = byte(c)
		}
	}
	special := func(pos int, set bool, lower, upper byte) {
		if !set {
			return
		}
		if b[pos] == 'x' {
			b[pos] = lower
		} else {
			b[pos] = upper
		}
	}
	special(3, m&fs.ModeSetuid != 0, 's', 'S')
	special(6, m&fs.ModeSetgid != 0, 's', 'S')
	special(9, m&fs.ModeSticky != 0, 't', 'T')
	return string(b)
}

func sortEntries(entries []Entry, by string, reverse, dirsFirst bool) {
	// "." and ".." always lead, as in ls.
	rank := func(e Entry) int {
		switch e.Name {
		case ".":
			return 0
		case "..":
			return 1
		}
		return 2
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if ra, rb := rank(a), rank(b); ra != rb || ra < 2 {
			return ra < rb
		}
		if dirsFirst && a.IsDir != b.IsDir {
			return a.IsDir
		}
		less := func() bool {
			switch by {
			case "size":
				if a.Size != b.Size {
					return a.Size > b.Size
				}
			case "time":
				if !a.modTime.Equal(b.modTime) {
					return a.modTime.After(b.modTime)
				}
			}
			return nameKey(a.Name) < nameKey(b.Name)
		}()
		if reverse {
			return !less
		}
		return less
	})
}

// nameKey sorts like ls in a typical UTF-8 locale: case-insensitive,
// ignoring a leading dot.
func nameKey(n string) string {
	return strings.ToLower(strings.TrimPrefix(n, "."))
}

func formatLong(entries []Entry, human, numeric bool) string {
	var total int64
	type row struct{ mode, links, owner, group, size, date, name string }
	rows := make([]row, len(entries))
	var wl, wo, wg, ws int
	for i, e := range entries {
		total += e.blocks
		owner, group := e.Owner, e.Group
		if numeric {
			owner, group = strconv.FormatUint(uint64(e.UID), 10), strconv.FormatUint(uint64(e.GID), 10)
		}
		size := strconv.FormatInt(e.Size, 10)
		if e.Device != "" {
			size = e.Device
		} else if human {
			size = humanSize(e.Size)
		}
		name := e.Name
		if e.Target != "" {
			name += " -> " + e.Target
		}
		rows[i] = row{e.Mode, strconv.FormatUint(e.Links, 10), owner, group, size, lsDate(e.modTime), name}
		wl, wo, wg, ws = max(wl, len(rows[i].links)), max(wo, len(owner)), max(wg, len(group)), max(ws, len(size))
	}
	var b strings.Builder
	// ls reports "total" in 1K blocks; st_blocks counts 512-byte units.
	totalStr := strconv.FormatInt(total/2, 10)
	if human {
		totalStr = humanSize(total * 512)
	}
	fmt.Fprintf(&b, "total %s\n", totalStr)
	for _, r := range rows {
		fmt.Fprintf(&b, "%s %*s %-*s %-*s %*s %s %s\n", r.mode, wl, r.links, wo, r.owner, wg, r.group, ws, r.size, r.date, r.name)
	}
	return b.String()
}

// lsDate formats like ls: "Sep 23 18:39" for the last six months,
// "Jun  6  2024" (year instead of time) for anything older or in the future.
func lsDate(t time.Time) string {
	if t.IsZero() {
		return "            "
	}
	now := time.Now()
	if t.After(now.AddDate(0, -6, 0)) && !t.After(now.Add(time.Hour)) {
		return t.Format("Jan _2 15:04")
	}
	return t.Format("Jan _2  2006")
}

// humanSize renders sizes like ls -h: 1023, 1.0K, 4.0K, 12K, 1.5M.
func humanSize(n int64) string {
	if n < 1024 {
		return strconv.FormatInt(n, 10)
	}
	v := float64(n)
	units := "KMGTPE"
	i := -1
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if v < 10 {
		// ls rounds up: 4097 bytes is 4.1K, never 4.0K.
		v = float64(int64(v*10+0.9999)) / 10
		if v < 10 {
			return strconv.FormatFloat(v, 'f', 1, 64) + string(units[i])
		}
	}
	return strconv.FormatInt(int64(v+0.9999), 10) + string(units[i])
}

// idNames maps ids to names from a passwd/group-format file (name:x:id:...),
// read once per listing instead of a lookup per entry.
func idNames(path string) map[uint32]string {
	names := map[uint32]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return names
	}
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Split(line, ":")
		if len(f) < 3 || strings.HasPrefix(line, "#") {
			continue
		}
		if id, err := strconv.ParseUint(f[2], 10, 32); err == nil {
			if _, dup := names[uint32(id)]; !dup {
				names[uint32(id)] = f[0]
			}
		}
	}
	return names
}

func nameOr(names map[uint32]string, id uint32) string {
	if n, ok := names[id]; ok {
		return n
	}
	return strconv.FormatUint(uint64(id), 10)
}
