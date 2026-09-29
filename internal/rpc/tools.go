package rpc

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"github.com/nucleusv/linux-mcp-daemon/internal/logging"
	sudorules "github.com/nucleusv/linux-mcp-daemon/internal/tools/auth/sudo-rules"
	systemcontrol "github.com/nucleusv/linux-mcp-daemon/internal/tools/kernel/system-control"
	"github.com/nucleusv/linux-mcp-daemon/internal/worker"
)

func (h *RPCHandler) HandleToolsList(session *Session, resp *JSONRPCResponse) {
	// One snapshot per request: a concurrent reload can't change the rules
	// between the authorization check and the call it authorizes.
	sudoCfg := h.Sudo()
	// Dynamically generate the tools list based on sudo rules.
	listDesc := "Lists the entries of ONE directory like `ls -l` (dotfiles, `.` and `..` are hidden unless `all: true`): type and permissions, link count, owner, group, size, modification time and symlink targets (symlinks are shown, never followed). Read-only, not recursive, no entry cap; `path` must be absolute. For a recursive or filtered search use `files/find`, for a directory's total size `disks/usage`, for a file's MIME type `files/filetype`. On permission denied retry with `privileged: true` if granted (root calls also need a `paths:` entry covering the path). Text output is `ls -l` lines (`Directory is empty.` when empty; `long: false` gives names only, directories suffixed `/`). `output_format: json` returns an array of objects (name, type, mode, mode_octal, links, owner, group, uid, gid, size, modified, is_dir, target), `[]` when empty."
	if sudoCfg.CanRunAsRoot(session.User, "files/list") {
		listDesc += " (Hint: You are authorized to run this tool as root. Use 'privileged: true' if you receive permission denied errors on sensitive paths)."
	}

	dfDesc := "Reports space on the ONE filesystem holding `path` (statfs, like `df` for a single path): total, used and free bytes and use percent (`human_readable` for GiB). Read-only; `path` must be absolute. It does not name the device or mount point: use `disks/mounts` for that, `disks/list` for block devices, `disks/usage` to see which folders use the space. There is no all-filesystems mode; call it once per mount point. `free` is what non-root users can use, and `used` is total minus that. `inodes: true` returns inode counts as plain text and ignores `output_format` and `human_readable`. Text output is three lines; `output_format: json` returns an object (path, total_bytes, used_bytes, free_bytes, use_percent, plus total_human, used_human, free_human with `human_readable`). `privileged: true` (a grant, and a `paths:` entry for root) only for paths you cannot stat."
	if sudoCfg.CanRunAsRoot(session.User, "disks/free") {
		dfDesc += " (Authorized for 'privileged: true')"
	}

	duDesc := "Measures how much disk space a directory tree uses (native walk, like `du`). Read-only. For the free space of a whole filesystem use `disks/free`; to find individual big files use `files/find` with `size`. By default the reply is one grand total; `max_depth: N` also lists directories up to N levels deep, largest first; `all: true` adds a per-file list (text output only). Sizes are allocated blocks unless `apparent_size: true`; hard links count once, symlinks are never followed. `exclude` patterns containing `/` match the full path (`/proc`, `/var/lib/*`), others the base name. Unreadable directories are skipped silently, so an unprivileged total can under-count: use `privileged: true` (a grant with a `paths:` entry). Results are cached for 60 s per user and arguments. The shipped config allows 300 s, otherwise the default is 30 s. Text ends with `Total size of PATH: N`; `output_format: json` returns an object (path, total_size, human_size with `human_readable`, directory_sizes as a path-to-bytes map only when `max_depth` > 0)."
	if sudoCfg.CanRunAsRoot(session.User, "disks/usage") {
		duDesc += " (Authorized for 'privileged: true' to traverse protected subdirectories)"
	}

	filetypeDesc := "Returns a file's MIME type, like `file -b --mime-type`, detected natively from its first 8 KiB (no file(1) needed); read-only, `path` must be absolute. A symlink is reported as `inode/symlink` (never followed); directories, devices, fifos and sockets as `inode/...`. Use it before `files/read`, which refuses binary files. For size, permissions or ownership use `files/list`; for contents `files/read`. The reply is always one plain-text line (no `output_format`). `privileged: true` (a grant, and for root a `paths:` entry) reads files your account cannot."
	if sudoCfg.CanRunAsRoot(session.User, "files/filetype") {
		filetypeDesc += " (Hint: You are authorized to run this tool as root. Use 'privileged: true' if you receive permission denied errors on sensitive paths)."
	}

	pkgDesc := "Lists installed packages by parsing the package database (dpkg on Debian/Ubuntu, apk on Alpine); rpm-based systems return an error, not supported yet. Read-only. The whole list is returned with no cap (hundreds of entries), so filter with `name`: an exact name or a glob (`openssh-*`, `*ssl*`). Only packages with status installed are listed. Text has a header `N packages installed (dpkg)` and a NAME VERSION ARCH table; `output_format: json` returns an array of objects (name, version, architecture), `[]` when none; there are no description or size fields. For OS and kernel version use `system/os-release`."
	if sudoCfg.CanRunAsRoot(session.User, "system/packages") {
		pkgDesc += " (Authorized for 'privileged: true' - when this daemon runs containerized, that automatically queries the real host's packages, not this container's own image.)"
	}

	mountsDesc := "Lists mounted filesystems (device, mount point, type, options) from /proc/thread-self/mounts, sorted by mount point. Read-only. `fs_type` filters by exact type (`ext4`, `overlay`, `tmpfs`; no globs). It reads the daemon's own mount namespace, so in a container use `privileged: true` (needs a grant) to get the host's mounts. Text lines look like `/dev/sda1 on /mnt type ext4 (rw,...)`; `output_format: json` returns an array of objects (device, mount_point, fs_type, options), or `null` rather than `[]` when nothing matches (text is then empty). For block devices use `disks/list`, for the space used on a mount `disks/free`."
	if sudoCfg.CanRunAsRoot(session.User, "disks/mounts") {
		mountsDesc += " (Authorized for 'privileged: true' - when this daemon runs containerized, that automatically shows the real host's mount table, not this container's own.)"
	}

	usersDesc := "Lists local user accounts from /etc/passwd and /etc/group (uid, gid, home, shell, supplementary group memberships), sorted by uid. Read-only. Never reads /etc/shadow - this reports account identity, not credentials. Local files only: LDAP/SSSD users are not listed. `min_uid` (e.g. 1000) hides system accounts. Text lines look like `name (uid=N gid=N(group)) home=... shell=... groups=a,b`; `output_format: json` returns an array of objects (username, uid, gid, group_name, comment, home_dir, shell, groups). For who logged in use `logs/logins`, for your own root grants `auth/sudo-rules`."
	if sudoCfg.CanRunAsRoot(session.User, "users/list") {
		usersDesc += " (Authorized for 'privileged: true' - when this daemon runs containerized, that automatically lists the real host's users, not this container's own.)"
	}

	loginsDesc := "Lists login history (wraps `last`) or failed login attempts (`type: \"failed\"`, wraps `lastb`). Read-only; returns raw text, not JSON. `limit` keeps the N most recent entries and `user` filters by username; the trailing `wtmp begins...` summary line is dropped and an empty history returns an empty line. `last`/`lastb` must be installed on the host. For account details use `users/list`, for authentication messages `logs/journal-control`."
	if sudoCfg.CanRunAsRoot(session.User, "logs/logins") {
		loginsDesc += " (Authorized for 'privileged: true' - typically required for type: \"failed\", since btmp is usually root-only readable. When this daemon runs containerized, privileged also automatically reads the real host's login history.)"
	}

	toolsList := map[string]interface{}{
		"tools": []interface{}{
			map[string]interface{}{
				"name":          "files/list",
				"tools_group":   "files",
				"linuxctl_verb": "list",
				"description":   listDesc,
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format":  map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
						"path":           map[string]interface{}{"type": "string", "description": "Absolute path of the directory to list"},
						"all":            map[string]interface{}{"type": "boolean", "description": "Include dotfiles, . and .. (ls -a)"},
						"long":           map[string]interface{}{"type": "boolean", "description": "Long listing like ls -l: type+permissions, links, owner, group, size, date, symlink target. Default true; false lists names only"},
						"human_readable": map[string]interface{}{"type": "boolean", "description": "Sizes like 4.0K, 1.5M (ls -h); default is bytes"},
						"sort":           map[string]interface{}{"type": "string", "enum": []string{"name", "size", "time"}, "description": "Sort by name (default), size (largest first) or time (newest first)"},
						"reverse":        map[string]interface{}{"type": "boolean", "description": "Reverse the sort order (ls -r)"},
						"dirs_first":     map[string]interface{}{"type": "boolean", "description": "List directories before files"},
						"numeric_ids":    map[string]interface{}{"type": "boolean", "description": "Show numeric uid/gid instead of names (ls -n)"},
						"privileged":     map[string]interface{}{"type": "boolean", "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused"},
					},
					"required": []string{"path"},
				},
			},
			map[string]interface{}{
				"name":          "files/read",
				"tools_group":   "files",
				"linuxctl_verb": "get",
				"description":   "Reads a text file and returns its contents as raw text (no line numbers or metadata). Read-only; `path` must be absolute. To find a file use `files/find`, for size or mode `files/list`, to check whether it is binary `files/filetype` (binary files are refused with `cannot read binary file`). Selection: `start_line`/`end_line` (1-indexed, inclusive; `start_line` alone reads to the end, `end_line` alone starts at line 1) take precedence over the byte range `offset`/`limit`. With no selection, or `offset` without `limit`, at most 10240 bytes come back followed by a `[WARNING: File truncated ...]` line, so page large files with `start_line`/`end_line` or pass `limit`. There is no streaming: one call reads into memory, and a line over 64 KiB fails line mode. A `start_line` past the end is an error; an empty file returns an empty string. `privileged: true` (needs a grant, and a `paths:` entry for root) reads files your account cannot.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":       map[string]interface{}{"type": "string", "description": "Absolute path of the file to read"},
						"start_line": map[string]interface{}{"type": "integer", "description": "First line to return (1-indexed). With or without end_line it takes precedence over offset/limit; alone it reads to the end of the file"},
						"end_line":   map[string]interface{}{"type": "integer", "description": "Last line to return (inclusive); alone it starts at line 1. Ignored if below start_line"},
						"offset":     map[string]interface{}{"type": "integer", "description": "Byte offset to start at; ignored when start_line/end_line is given. Without limit at most 10240 bytes are returned"},
						"limit":      map[string]interface{}{"type": "integer", "description": "Number of bytes to return (no upper cap). With no selection at all, the first 10240 bytes are returned"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused"},
					},
					"required": []string{"path"},
				},
			},
			map[string]interface{}{
				"name":          "files/create",
				"tools_group":   "files",
				"linuxctl_verb": "create",
				"description":   "Writes a whole file: creates it, or REPLACES the content of an existing one entirely. Mutating and not atomic. Missing parent directories are created (mode 0755); a new file gets mode 0644, an existing file keeps its mode. With `content` omitted or empty it only touches: creates an empty file or refreshes the modification time and never truncates an existing file. To change part of an existing file use `files/update` (append or replace lines); to check first whether a path exists use `files/list` or `files/find`; afterwards set mode or owner with `files/chmod`/`files/chown`. `content` is text and no trailing newline is added; `path` must be absolute. Writing where your account cannot needs `privileged: true` (a grant, and for root a `paths:` entry covering the path, which also refuses symlinks in the path; otherwise a symlink at the path is followed). Returns one line, `Successfully created and wrote to PATH` or `Successfully touched PATH`; failures are plain text such as `failed to write to file`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":       map[string]interface{}{"type": "string", "description": "Absolute path of the file; missing parent directories are created"},
						"content":    map[string]interface{}{"type": "string", "description": "Text to write; replaces any existing content entirely, no newline is added. Omit or leave empty to only create an empty file or refresh its mtime (never truncates)"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused"},
					},
					"required": []string{"path"},
				},
			},
			map[string]interface{}{
				"name":          "files/update",
				"tools_group":   "files",
				"linuxctl_verb": "update",
				"description":   "Edits part of an EXISTING file in place: appends text or replaces an inclusive line range. Mutating and not atomic; the file keeps its mode and owner. To write a whole file use `files/create`; to see the lines first use `files/read` with `start_line`/`end_line`. With `append: true` exactly `content` is added at the end (no newline is added, include your own) and the file is created if missing, though not its parent directories; append wins over any line range. Otherwise `start_line` AND `end_line` are both required (1-indexed, `end_line` >= `start_line`), the file must exist, and those lines are replaced by `content` (one trailing newline of `content` is ignored; an `end_line` past the end is clamped; a `start_line` past the end adds `content` as a new last line). There is no insert or delete mode: replacing with empty `content` leaves one empty line. Returns `Successfully appended to PATH` or `Successfully updated lines A-B in PATH`, no diff. `path` must be absolute; `privileged: true` (a grant, and for root a `paths:` entry) edits files you cannot write.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":       map[string]interface{}{"type": "string", "description": "Absolute path of the file to edit"},
						"content":    map[string]interface{}{"type": "string", "description": "Text to append, or to replace the line range with (no newline is added when appending)"},
						"append":     map[string]interface{}{"type": "boolean", "description": "If true, append content at the end (creates the file if missing); takes precedence over start_line/end_line"},
						"start_line": map[string]interface{}{"type": "integer", "description": "First line to replace (1-indexed); required together with end_line unless append is true"},
						"end_line":   map[string]interface{}{"type": "integer", "description": "Last line to replace (inclusive, >= start_line; past the end of the file is clamped)"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused"},
					},
					"required": []string{"path", "content"},
				},
			},
			map[string]interface{}{
				"name":          "files/find",
				"tools_group":   "files",
				"linuxctl_verb": "find",
				"description":   "Searches a directory tree (default `/`) by name, type, age or size, like `find`. Read-only; never follows symlinks; skips `/proc`, `/sys`, `/dev` and `/run` when the search starts at `/` or above them (not when `path` is inside one). Use `files/list` to see one known directory and `disks/usage` to see which folders take the space. All filters are ANDed and none is required. There is NO result cap: `path: /` without a filter lists every file on the host, so give `name`, `type` or `max_depth`; the 30 s worker timeout applies. Syntax: `name` is a case-sensitive glob on the base name only (`*.log`, not a path); `type` is one of `f d l b c p s` or a comma list (`f,d`); `mtime` in days: `+7` older than 7 days, `-1` within the last day, `7` exactly 7 days; `size`: `+100M` larger, `-10k` smaller (units b c w k M G, rounded up); `max_depth` 1 = direct children, omitted or 0 = unlimited. Unreadable directories are skipped silently. Text output: one `SIZE PATH` line per match, sorted by path (no size for directories; empty output = no match). `output_format: json` returns an array of objects (path, size in bytes, type).",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"human_readable": map[string]interface{}{"type": "boolean", "description": "Text output only: sizes like 1.5 KiB; default is bytes"},
						"output_format":  map[string]interface{}{"type": "string", "description": "json (yaml, table and wide return the same JSON) gives an array of objects with path, size, type; default is text"},
						"path":           map[string]interface{}{"type": "string", "description": "Absolute starting directory (default /). A large tree with no filter can take longer than the 30 s worker limit"},
						"name":           map[string]interface{}{"type": "string", "description": "Glob on the file's base name only, case-sensitive, e.g. '*.log' (not a path pattern)"},
						"type":           map[string]interface{}{"type": "string", "description": "f file, d directory, l symlink, b, c, p, s; or a comma list such as 'f,d'"},
						"mtime":          map[string]interface{}{"type": "string", "description": "Days since modification: '+7' older than 7 days, '-1' within the last day, '7' exactly 7 days"},
						"size":           map[string]interface{}{"type": "string", "description": "'+100M' larger than 100 MiB, '-10k' smaller than 10 KiB; units b c w k M G, rounded up to the unit"},
						"max_depth":      map[string]interface{}{"type": "integer", "description": "Levels below path to descend (1 = direct children); omit or 0 for unlimited"},
						"privileged":     map[string]interface{}{"type": "boolean", "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused"},
					},
					"required": []string{},
				},
			},
			map[string]interface{}{
				"name":          "files/filetype",
				"tools_group":   "files",
				"linuxctl_verb": "filetype",
				"description":   filetypeDesc,
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":       map[string]interface{}{"type": "string", "description": "Absolute path to the file"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused"},
					},
					"required": []string{"path"},
				},
			},
			map[string]interface{}{
				"name":          "files/chmod",
				"tools_group":   "files",
				"linuxctl_verb": "chmod",
				"description":   "Changes a file's or directory's permission bits (chmod). Mutating and idempotent; for owner or group use `files/chown`, to check the result `files/list`. Never follows symbolic links: a path containing a symlink in any component is refused, and recursive changes skip symlinks and report them. Numeric modes follow GNU chmod semantics (on directories a 4-digit mode keeps setuid/setgid; use 5 digits, e.g. 00755, to set them exactly); a bare `755` is octal. Changing a file you do not own needs `privileged: true` (a grant, and a `paths:` entry for root). Single change returns `PATH: 0644 (-rw-r--r--) -> 0755 (-rwxr-xr-x)`, or `... unchanged` if already set. A recursive run prints one line per changed entry, then `changed N, unchanged M` (plus skipped symlinks) and per-entry errors; it is not atomic, so partial success is possible.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":       map[string]interface{}{"type": "string", "description": "Absolute path"},
						"mode":       map[string]interface{}{"type": "string", "description": "Octal (644, 0755, 4755) or symbolic (u+x, go-w, a=r, +X, u+s, +t; comma-separated)"},
						"recursive":  map[string]interface{}{"type": "boolean", "description": "Also apply to everything below a directory (symlinks are skipped, never followed)"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root - needed for files you don't own. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path"},
					},
					"required": []string{"path", "mode"},
				},
			},
			map[string]interface{}{
				"name":          "files/chown",
				"tools_group":   "files",
				"linuxctl_verb": "chown",
				"description":   "Changes a file's or directory's owner and/or group (chown). Mutating and idempotent; for permission bits use `files/chmod`, to check the result `files/list`. Never follows symbolic links: a path containing a symlink in any component is refused, and recursive changes skip symlinks and report them. Changing the owner requires `privileged: true` (a grant, and a `paths:` entry for root). `owner` is `user`, `user:group`, `:group` or `user:` (the user's login group), with names or numeric ids; names are looked up in the host's /etc/passwd and /etc/group and an unknown one fails (`no such user`). Returns `PATH: old -> new` as owner:group, or `unchanged`; a recursive run prints one line per changed entry then a summary with skipped symlinks and per-entry errors, and is not atomic.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":       map[string]interface{}{"type": "string", "description": "Absolute path"},
						"owner":      map[string]interface{}{"type": "string", "description": "user, user:group, :group, or user: (the user's login group); names or numeric ids"},
						"recursive":  map[string]interface{}{"type": "boolean", "description": "Also apply to everything below a directory (symlinks are skipped, never followed)"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root - required to change ownership. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path"},
					},
					"required": []string{"path", "owner"},
				},
			},
			map[string]interface{}{
				"name":          "disks/free",
				"tools_group":   "disks",
				"linuxctl_verb": "free",
				"description":   dfDesc,
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format":  map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
						"path":           map[string]interface{}{"type": "string", "description": "Absolute path to check"},
						"inodes":         map[string]interface{}{"type": "boolean", "description": "Report inode counts instead of block usage (-i); the reply is always plain text and ignores output_format and human_readable"},
						"human_readable": map[string]interface{}{"type": "boolean", "description": "Sizes like 53.2 GiB (df -h); default is bytes"},
						"privileged":     map[string]interface{}{"type": "boolean", "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused"},
					},
					"required": []string{"path"},
				},
			},
			map[string]interface{}{
				"name":          "disks/usage",
				"tools_group":   "disks",
				"linuxctl_verb": "usage",
				"description":   duDesc,
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"human_readable":  map[string]interface{}{"type": "boolean", "description": "Sizes like du -h (4.0 KiB); default is bytes"},
						"output_format":   map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
						"path":            map[string]interface{}{"type": "string", "description": "Absolute path of the directory to measure"},
						"max_depth":       map[string]interface{}{"type": "integer", "description": "0 or omitted: grand total only; N: also list directories up to N levels deep, largest first"},
						"one_file_system": map[string]interface{}{"type": "boolean", "description": "Skip directories on different file systems (-x)"},
						"exclude":         map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Patterns to skip: one containing '/' matches the full path (e.g. '/proc', '/var/lib/*'), others the base name (e.g. '*.tmp')"},
						"all":             map[string]interface{}{"type": "boolean", "description": "Also list every file, largest first (text output only)"},
						"apparent_size":   map[string]interface{}{"type": "boolean", "description": "Report logical file sizes instead of allocated disk blocks"},
						"threshold":       map[string]interface{}{"type": "integer", "description": "Bytes: a positive value hides entries smaller than this, a negative value hides larger ones (printed lines only)"},
						"separate_dirs":   map[string]interface{}{"type": "boolean", "description": "For directories do not include size of subdirectories (-S)"},
						"privileged":      map[string]interface{}{"type": "boolean", "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused"},
					},
					"required": []string{"path"},
				},
			},
			map[string]interface{}{
				"name":          "processes/top",
				"tools_group":   "processes",
				"linuxctl_verb": "top",
				"description":   "Snapshot like `top -b -n 1`, read from `/proc`: header (uptime, users, load average, task counts by state, CPU us/sy/ni/id/wa/hi/si/st, memory and swap) plus the process table (PID USER PR NI VIRT RES SHR S %CPU %MEM TIME+ COMMAND), sorted by `sort_by` (default cpu). Read-only. %CPU is measured over `interval_ms` (default 1000, max 10000), so the call takes about that long. For a lighter PID list or a single PID use `processes/list`; to signal a process `processes/delete`; for memory totals only `memory/usage`; for which process owns a port `network/connections`. `limit` defaults to all processes; `user` is an exact username. Sizes are bytes (MiB with `human_readable`). `output_format`: default and `table` give top's layout, `wide` adds PPID, THR and full command lines, `json`/`yaml` return an object with `summary` and `processes` (memory in bytes).",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"human_readable": map[string]interface{}{"type": "boolean", "description": "Memory like top (MiB header, m/g columns); default is bytes"},
						"sort_by":        map[string]interface{}{"type": "string", "enum": []string{"cpu", "mem", "res", "time", "pid"}, "description": "Sort column: cpu (default, like top), mem/res (resident memory), time (total CPU time), pid"},
						"limit":          map[string]interface{}{"type": "integer", "description": "Maximum processes to list (default: all)"},
						"user":           map[string]interface{}{"type": "string", "description": "Only this user's processes"},
						"interval_ms":    map[string]interface{}{"type": "integer", "description": "%CPU sampling interval in milliseconds (default 1000, max 10000)"},
						"output_format":  map[string]interface{}{"type": "string", "description": "Default/table: top's own layout. wide: adds PPID, THR and full command lines (like top -c). json/yaml: structured {summary, processes}, memory in bytes (mem_bytes, swap_bytes, virt_bytes, res_bytes, shr_bytes)"},
						"privileged":     map[string]interface{}{"type": "boolean", "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused"},
					},
				},
			},
			map[string]interface{}{
				"name":          "processes/list",
				"tools_group":   "processes",
				"linuxctl_verb": "get",
				"description":   "Lists processes (PID, PPID, user, state, RSS, command) read from `/proc`. Read-only. Use it to find a PID, filter by `user` or one `pid`, or sort by memory. For the CPU/memory header and %CPU on every row use `processes/top`; for the process that owns a port `network/connections`; to signal a process `processes/delete`; for deep per-PID metrics the `process://<pid>/<target>` resource. Sorted by PID unless `sort_by` is `mem` (RSS, largest first) or `cpu` (samples for 0.5 s, so the call takes at least that long, and only then does `cpu_percent` appear); `limit` applies after sorting. Kernel threads appear as `[name]`. Command lines can contain secrets passed as arguments. Text output is a table; `output_format: json` returns an array of objects (pid, user, comm, state, ppid, rss_bytes, cmdline, cpu_percent). Sizes are bytes unless `human_readable`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"human_readable": map[string]interface{}{"type": "boolean", "description": "RSS like 10Mi; default is bytes"},
						"output_format":  map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
						"user":           map[string]interface{}{"type": "string", "description": "Exact username"},
						"pid":            map[string]interface{}{"type": "integer", "description": "Filter to a single specific PID"},
						"sort_by":        map[string]interface{}{"type": "string", "description": "pid (default), mem (RSS, largest first) or cpu (samples for 0.5 s)"},
						"limit":          map[string]interface{}{"type": "integer", "description": "Maximum rows returned, applied after sorting"},
						"privileged":     map[string]interface{}{"type": "boolean", "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused"},
					},
				},
			},
			map[string]interface{}{
				"name":          "processes/delete",
				"tools_group":   "processes",
				"linuxctl_verb": "delete",
				"description":   "Sends ONE signal to ONE process by PID (kill(2)); the default SIGTERM asks the process to exit. Mutating and not idempotent: it returns as soon as the signal is delivered and does not check that the process exited, and signalling a PID that is gone fails with `no such process`. Allowed `signal` values: SIGTERM, SIGKILL, SIGHUP, SIGINT, SIGQUIT, SIGUSR1, SIGUSR2, SIGSTOP, SIGCONT, SIGABRT (SIG prefix optional, case-insensitive, numbers rejected), so it can also pause (SIGSTOP) and resume (SIGCONT). Refuses PID 1 and the mcpd daemon itself. Another user's process fails with `not permitted` unless `privileged: true` (needs a grant). Find the PID first with `processes/list` or `processes/top`. To stop a managed service use `services/manage` (systemd may restart a killed one), for a container `docker/manage`. Returns one text line, `Successfully sent signal SIGTERM to process N`; `output_format` has no effect.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Ignored - the reply is always text"},
						"pid":           map[string]interface{}{"type": "integer", "description": "PID to signal (positive integer; PID 1 and mcpd itself are refused)"},
						"signal":        map[string]interface{}{"type": "string", "description": "SIGTERM (default), SIGKILL, SIGHUP, SIGINT, SIGQUIT, SIGUSR1, SIGUSR2, SIGSTOP, SIGCONT or SIGABRT; SIG prefix optional, case-insensitive"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root to signal other users' processes. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused"},
					},
					"required": []string{"pid"},
				},
			},
			map[string]interface{}{
				"name":          "network/nslookup",
				"tools_group":   "network",
				"linuxctl_verb": "nslookup",
				"description":   "Resolves DNS records for a hostname through the host's resolver (/etc/resolv.conf; a DNS server cannot be chosen), so answers may come from a local cache. Read-only, but it sends queries off-host. `record_type` (case-insensitive) is A, AAAA, CNAME, TXT, MX, NS or ANY; the default ANY is not a DNS ANY query but runs the CNAME, A/AAAA, TXT, MX and NS lookups in turn. Other types (SOA, PTR, SRV) are rejected, and `host` must be a name: an IP address is not reverse-resolved. Always returns JSON: an object with `host` and `records` (array of objects with `type` and `value`; an MX value looks like `10 mail.example.com.`). Empty `records` means the name exists but has no record of that type; a name that does not exist is an error (`HOST: no such host`). To test reachability use `network/ping`, to fetch a URL `network/curl`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"host":        map[string]interface{}{"type": "string", "description": "Hostname to resolve, e.g. example.com (an IP address is not reverse-resolved)"},
						"record_type": map[string]interface{}{"type": "string", "description": "A, AAAA, CNAME, TXT, MX, NS or ANY (default ANY: all of these are looked up in turn)"},
					},
					"required": []string{"host"},
				},
			},
			map[string]interface{}{
				"name":          "network/curl",
				"tools_group":   "network",
				"linuxctl_verb": "curl",
				"description":   "Makes one HTTP(S) request with Go's HTTP client and returns status, headers and body. NOT read-only: any `method` (default GET) is sent as given, so POST, PUT or DELETE change the remote system. Follows redirects (up to 10); a non-2xx status is not an error, check `status_code`. The default timeout is 10 s (`timeout`, whole seconds) and the 30 s worker limit caps anything larger. The body is cut at `max_body` (default 1 MiB, max 10 MiB) and `truncated` is then true. `insecure` skips TLS verification. The daemon's proxy environment is honored unless the user has a `network:` policy in mcp-sudo.yaml; such a policy applies to every call and redirect hop, and a blocked destination fails to connect. Header and body values are redacted in the audit log. Returns JSON with `status_code`, `status`, `headers` (values comma-joined), `body`, `truncated`; `output_format` is ignored. For DNS use `network/nslookup`, for TCP reachability `network/ping`, for local files `files/read`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"url":      map[string]interface{}{"type": "string", "description": "Full URL including scheme, e.g. https://example.com/api"},
						"method":   map[string]interface{}{"type": "string", "description": "HTTP method, default GET; any method is sent as given"},
						"body":     map[string]interface{}{"type": "string", "description": "Request body"},
						"headers":  map[string]interface{}{"type": "object", "description": "Request headers, e.g. {\"Content-Type\": \"application/json\"}", "additionalProperties": map[string]interface{}{"type": "string"}},
						"insecure": map[string]interface{}{"type": "boolean", "description": "Skip TLS certificate verification"},
						"timeout":  map[string]interface{}{"type": "number", "description": "Whole seconds (default 10; the 30 s worker limit caps it)"},
						"max_body": map[string]interface{}{"type": "integer", "description": "Return at most this many bytes of the response body (default 1048576 = 1 MiB, max 10 MiB); a cut body has truncated: true"},
					},
					"required": []string{"url"},
				},
			},
			map[string]interface{}{
				"name":          "network/arp",
				"tools_group":   "network",
				"linuxctl_verb": "arp",
				"description":   "Shows the kernel's ARP cache (IPv4 address to MAC address) from /proc/net/arp in the daemon's network namespace. Read-only. It is a cache, not a scan: only hosts contacted recently appear, and IPv6 neighbours are not included. `interface` is an exact device name (`eth0`); omit it for all. For sockets and connections use `network/connections`, for reachability `network/ping`. Always returns JSON: an array of objects (ip_address, hw_type and flags as raw hex such as `0x1`, hw_address, mask, device). When nothing matches the output is `null`, not `[]`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"interface": map[string]interface{}{"type": "string", "description": "Exact interface name such as eth0; omit for all interfaces"},
					},
				},
			},
			map[string]interface{}{
				"name":          "network/ping",
				"tools_group":   "network",
				"linuxctl_verb": "ping",
				"description":   "Tests TCP reachability: opens one TCP connection to `host:port` and closes it. This is NOT ICMP, so it needs a listening port (default 80) and says nothing about other ports or ICMP. Single attempt, no loss statistics; latency includes DNS resolution. Read-only, but it connects off-host and honors the user's `network:` policy in mcp-sudo.yaml. `timeout` is whole seconds (default 5). A failed connect is not a tool error: the JSON has `success: false` and an `error` text. Returns JSON with host, port, success, latency_ms and error. For DNS only use `network/nslookup`, for the hop path `network/trace-path`, for an HTTP check `network/curl`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"host":    map[string]interface{}{"type": "string", "description": "Hostname or IP address to connect to"},
						"port":    map[string]interface{}{"type": "number", "description": "TCP port, whole number (default 80)"},
						"timeout": map[string]interface{}{"type": "number", "description": "Seconds, whole number (default 5)"},
					},
					"required": []string{"host"},
				},
			},
			map[string]interface{}{
				"name":          "network/connections",
				"tools_group":   "network",
				"linuxctl_verb": "connections",
				"description":   "Lists TCP and UDP sockets in every state with their owning processes, like `ss -tuanp`, read natively from /proc/net (no ss needed). Read-only; unix and raw sockets are not included. `state` (case-insensitive, `_` and `-` interchangeable) filters by LISTEN (includes unconnected UDP), ESTABLISHED, TIME_WAIT, CLOSE_WAIT, SYN_SENT, ... or the groups connected/synchronized; an invalid state is an error listing the valid names. `port` matches the local OR peer port. The owning process (pid, fd) is shown only for sockets whose /proc/PID/fd you can read; other users' processes need `privileged: true` (needs a grant). Text output has `ss -tuanp` columns; `output_format: json` returns an array of objects (netid, state, recv_q, send_q, local_address, local_port, peer_address, peer_port, uid, inode, processes), `[]` when empty. For the ARP cache use `network/arp`, for interfaces and IPs the `network://interfaces` resource, for process details `processes/list`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
						"state":         map[string]interface{}{"type": "string", "description": "Filter by TCP state, case-insensitive (LISTEN/listening, ESTABLISHED, TIME_WAIT, CLOSE_WAIT, SYN_SENT, ...). Omit to list all sockets - active connections and listening ports."},
						"port":          map[string]interface{}{"type": "integer", "description": "Match sockets whose local or peer port equals this"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root to see PIDs of other users. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused"},
					},
				},
			},
			map[string]interface{}{
				"name":          "memory/usage",
				"tools_group":   "memory",
				"linuxctl_verb": "usage",
				"description":   "Reports memory and swap use from /proc/meminfo. Read-only. `used` is total - free - (buffers + cached + reclaimable slab); `available` is the kernel's MemAvailable. Text output is a `free`-like table with Mem: and Swap: rows (bytes, or e.g. `1.8Gi` with `human_readable`); `detailed: true` returns the raw /proc/meminfo instead, but only for text output (it is ignored with `output_format: json`). JSON (also yaml/table/wide) returns an object (total, used, free, shared, buffCache, available, swap_total, swap_used, swap_free) in bytes; `human_readable` is ignored there. For per-process memory use `processes/top` or `processes/list`, for CPU load `cpu/load-average`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"human_readable": map[string]interface{}{"type": "boolean", "description": "Sizes like free -h (1.8Gi); default is bytes"},
						"output_format":  map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
						"detailed":       map[string]interface{}{"type": "boolean", "description": "Return the raw /proc/meminfo instead of the summary (text output only; ignored with output_format json)"},
					},
				},
			},
			map[string]interface{}{
				"name":          "services/manage",
				"tools_group":   "system",
				"linuxctl_verb": "services",
				"description":   "Starts, stops, restarts, reloads, enables or disables ONE systemd service over D-Bus (`.service` is appended when missing). Mutating. Ordinary users are usually refused by polkit, so `privileged: true` (needs a grant in mcp-sudo.yaml) is normally required. start, stop, restart and reload wait for the systemd job and return `Job N completed with status: done` (or failed, canceled, timeout, dependency, skipped); a slow one can hit the 30 s worker limit. `reload` asks the service to re-read its config without stopping it, only if the unit supports it. `enable` and `disable` only change whether it starts at boot; they do not start or stop it. To see state use `services/list` or the `service://<name>/status` resource, for logs `logs/journal-control`, for containers `docker/manage`, for a raw signal to a PID `processes/delete`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"service":    map[string]interface{}{"type": "string", "description": "Unit name; '.service' is appended if missing (e.g. 'kubelet' or 'kubelet.service')"},
						"action":     map[string]interface{}{"type": "string", "enum": []string{"start", "stop", "restart", "reload", "enable", "disable"}, "description": "start, stop, restart and reload wait for the systemd job; enable and disable only change start at boot"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root. Normally required (polkit); needs a grant for this tool in mcp-sudo.yaml"},
					},
					"required": []string{"service", "action"},
				},
			},
			map[string]interface{}{
				"name":          "services/list",
				"tools_group":   "system",
				"linuxctl_verb": "services",
				"description":   "Lists systemd `.service` units that systemd currently has loaded, with load, active and sub state (D-Bus ListUnits). Read-only. An installed but never-loaded unit file may be missing, and timers, sockets and other unit types are not included. `pattern` supports only a leading and/or trailing `*` (`kube*`, `*ssh*`); without `*` it is an exact unit name including `.service`. The state filters are exact strings: `active_state` active/failed/inactive, `sub_state` running/exited/dead, `load_state` loaded/not-found. Text output is a block per unit plus a hint line, or `No services found matching the criteria.`; `output_format: json` returns an array of objects (name, description, load_state, active_state, sub_state), `[]` when empty. To change a service use `services/manage`, for one unit's details the `service://<name>/status` resource, for its logs `logs/journal-control`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pattern":       map[string]interface{}{"type": "string", "description": "Only a leading and/or trailing * is supported ('kube*', '*ssh*'); without * an exact unit name including '.service'"},
						"active_state":  map[string]interface{}{"type": "string", "description": "Exact state: 'active', 'failed' or 'inactive'"},
						"load_state":    map[string]interface{}{"type": "string", "description": "Exact state: 'loaded' or 'not-found'"},
						"sub_state":     map[string]interface{}{"type": "string", "description": "Exact state: 'running', 'exited' or 'dead'"},
						"output_format": map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root (may be required depending on policies)"},
					},
					"required": []string{},
				},
			},
			map[string]interface{}{
				"name":          "timers/list",
				"tools_group":   "timers",
				"linuxctl_verb": "get",
				"description":   "Lists the systemd timers of the host - systemd's scheduler, the modern counterpart of cron - each with the unit it starts, its schedule (`OnCalendar=` and monotonic settings such as `OnBootSec=`), its next and last run, whether it is `Persistent` (catches up runs missed while the host was off) and its last result. Read-only. Use it to answer \"what runs on a schedule, and when next\": `services/list` shows only `.service` units, so timers never appear there. To inspect the unit a timer starts, use `linuxctl describe system <name>` or the `service://<name>/status` resource; a timer's own state is in this listing. Times are RFC 3339 UTC, and `never` means systemd reports none (a timer that has not fired yet, or one with no upcoming trigger). `pattern` accepts a leading and/or trailing `*` (`apt*`, `*.timer`, `*daily*`); the timer name includes `.timer`. Unprivileged calls need the host's systemd bus (a bare-metal or VM host); inside a container use `privileged: true`, which joins the host. `output_format: json` returns an array with the same fields; text is the default. Empty result means no timer matched. Managing timers (start, stop, enable) is not offered by this tool.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pattern":       map[string]interface{}{"type": "string", "description": "Wildcard on the timer name (e.g. 'apt*', '*.timer', '*daily*'); no other wildcards"},
						"active_state":  map[string]interface{}{"type": "string", "description": "Only timers in this active state (e.g. 'active', 'inactive', 'failed')"},
						"output_format": map[string]interface{}{"type": "string", "description": "'json' (also yaml/table/wide, which return the same JSON) for an array of objects; default is text"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root (needed inside a container to reach the host's systemd; needs a grant)"},
					},
					"required": []string{},
				},
			},
			map[string]interface{}{
				"name":          "logs/journal-control",
				"tools_group":   "logs",
				"linuxctl_verb": "journal",
				"description":   "Reads the systemd journal (wraps `journalctl -n LINES --no-pager`). Read-only. Returns the last `lines` entries (default 100, the only size limit), oldest first unless `reverse`. Filter with `unit` (`sshd.service`), `since`/`until` in journalctl syntax (`1 hour ago`, `yesterday`, `2026-09-29 10:00`), `boot: true` (current boot) or `boot_offset` (-1 = previous boot; takes precedence over `boot`). Without root an ordinary user sees only their own entries unless in the `systemd-journal` or `adm` group. Requires privileged: true in containerized deployments (needs a grant), since journalctl only exists on the host, never in this daemon's own image. Plain text by default; `output_format: json` gives one JSON object per line (journalctl -o json), not an array. For kernel messages use `logs/dmesg`, for logins `logs/logins`, for container output `docker/logs`, for unit state `services/list`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"unit":          map[string]interface{}{"type": "string", "description": "Filter by systemd unit (e.g., 'kubelet.service')"},
						"lines":         map[string]interface{}{"type": "integer", "description": "Number of entries to tail (default 100; the only size limit)"},
						"since":         map[string]interface{}{"type": "string", "description": "journalctl time syntax, e.g. '1 hour ago', 'today', '2026-09-29 10:00'"},
						"until":         map[string]interface{}{"type": "string", "description": "Filter logs until a specific time (e.g., 'yesterday', '12:00')"},
						"reverse":       map[string]interface{}{"type": "boolean", "description": "Output newest entries first"},
						"boot":          map[string]interface{}{"type": "boolean", "description": "Restrict output to the current boot (journalctl -b)"},
						"boot_offset":   map[string]interface{}{"type": "integer", "description": "Select a prior boot relative to the current one, e.g. -1 for the previous boot (implies boot)"},
						"output_format": map[string]interface{}{"type": "string", "description": "json returns one JSON object per line (journalctl -o json); default is plain text"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root and join the host mount namespace - required in containerized deployments (needs a grant for this tool in mcp-sudo.yaml)"},
					},
				},
			},
			map[string]interface{}{
				"name":          "logs/dmesg",
				"tools_group":   "logs",
				"linuxctl_verb": "dmesg",
				"description":   "Reads the kernel ring buffer (wraps `dmesg --human`, relative timestamps). Read-only. Output is cut to the LAST 30 KiB with a `[WARNING: Output truncated to last 30KB]` prefix and cannot be paged; narrow it with `level`, a comma list of emerg, alert, crit, err, warn, notice, info, debug (`err,warn`). On hosts with `kernel.dmesg_restrict=1` an unprivileged call fails (dmesg's error is returned); use `privileged: true` (needs a grant). `output_format` is accepted and ignored: the reply is always plain text. For older history or service logs use `logs/journal-control`, for login records `logs/logins`, for drive faults `disks/health`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"level":         map[string]interface{}{"type": "string", "description": "Comma-separated levels from emerg, alert, crit, err, warn, notice, info, debug (e.g. 'err,warn')"},
						"output_format": map[string]interface{}{"type": "string", "description": "Ignored - the reply is always plain text"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root - needed when kernel.dmesg_restrict=1. Needs a grant for this tool in mcp-sudo.yaml"},
					},
				},
			},
			map[string]interface{}{
				"name":          "logs/logins",
				"tools_group":   "logs",
				"linuxctl_verb": "logins",
				"description":   loginsDesc,
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"type":       map[string]interface{}{"type": "string", "enum": []string{"success", "failed"}, "description": "\"success\" (default, wraps `last`) or \"failed\" (wraps `lastb`)"},
						"limit":      map[string]interface{}{"type": "integer", "description": "Only return this many most recent entries"},
						"user":       map[string]interface{}{"type": "string", "description": "Only return entries for this username"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root - typically required for type \"failed\". Needs a grant for this tool in mcp-sudo.yaml"},
					},
				},
			},
			map[string]interface{}{
				"name":          "kernel/system-control",
				"tools_group":   "kernel",
				"linuxctl_verb": "sysctl",
				"description":   "Reads or writes a kernel parameter (sysctl) at runtime through /proc/sys. With `value` omitted it reads: `key` (dotted `net.ipv4.ip_forward` or slash form; a directory such as `net.ipv4` prints its subtree) returns `key = value` lines, and `read_all: true` (only when `key` is empty) prints every parameter, thousands of lines, uncapped. Reading needs no grant. With `value` it WRITES: that needs `privileged: true` and a grant, is refused for keys outside the user's `sysctl.write_keys` globs in mcp-sudo.yaml, and the reply shows the value the kernel now holds. Writes last until reboot; nothing is persisted to /etc/sysctl.d. For OS and kernel version use `system/os-release`, for memory figures `memory/usage`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"key":        map[string]interface{}{"type": "string", "description": "Kernel parameter name, dotted (net.ipv4.ip_forward) or slash form (net/ipv4/conf/eth0.100/rp_filter). A directory (e.g. net.ipv4) reads its whole subtree."},
						"value":      map[string]interface{}{"type": "string", "description": "Value to write (single line). If omitted, the parameter is read. Writing needs privileged: true and a matching sysctl.write_keys grant"},
						"read_all":   map[string]interface{}{"type": "boolean", "description": "Only when key is empty: read every parameter (thousands of lines, uncapped)"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root - required for writes. Needs a grant for this tool in mcp-sudo.yaml"},
					},
				},
			},
			map[string]interface{}{
				"name":          "cpu/list",
				"tools_group":   "cpu",
				"linuxctl_verb": "get",
				"description":   "Lists the machine's CPUs from /proc/cpuinfo. Read-only. Text output shows the number of logical processors and, for the FIRST processor only, vendor, model name, MHz (or BogoMIPS) and cache size. `output_format: json` (also yaml/table/wide) returns an array with one object per logical CPU using /proc/cpuinfo's own field names, which differ by architecture (x86 `model name`, `cpu MHz`, `flags`; ARM `CPU implementer`). It does not report sockets, cores or threads separately, and `topology_only` has no effect. For current load use `cpu/load-average`, for per-process CPU `processes/top`, for OS and kernel `system/os-release`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
						"topology_only": map[string]interface{}{"type": "boolean", "description": "Has no effect: accepted but ignored, the output is the same"},
					},
				},
			},
			map[string]interface{}{
				"name":          "cpu/load-average",
				"tools_group":   "cpu",
				"linuxctl_verb": "load-average",
				"description":   "Returns the 1, 5 and 15 minute load averages from sysinfo(2). Read-only; the only parameter is `output_format`. Load counts runnable plus uninterruptible tasks, not CPU percent: compare it with the number of logical CPUs from `cpu/list`. Text is `Load Average: 0.52, 0.48, 0.45`; `output_format: json` (also yaml/table/wide) returns an object with numbers `1_min`, `5_min`, `15_min`. For per-process CPU use `processes/top`, for memory `memory/usage`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"}},
				},
			},
			map[string]interface{}{
				"name":          "disks/list",
				"tools_group":   "disks",
				"linuxctl_verb": "get",
				"description":   "Lists block devices as a tree (like `lsblk`): disks, partitions, and LVM/dm-crypt/RAID volumes nested under the devices they are built on, with MAJ:MIN, RM, SIZE, RO, TYPE and MOUNTPOINTS, read from /sys/class/block (no lsblk needed). Read-only. Empty devices and RAM disks are hidden unless `all: true`; SIZE is in bytes unless `human_readable`. `output_format: json` (also yaml/table/wide) returns an object `blockdevices`, an array of objects (name, kname, maj:min, rm, size, size_bytes, ro, type, mountpoints) with nested `children`. In containerized deployments use `privileged: true` (needs a grant) to see the host's mount points. For free space or inodes use `disks/free`, for folder sizes `disks/usage`, for the mount table `disks/mounts`, for partition boundaries `disks/partitions`, for I/O counters `disks/performance`, for SMART `disks/health`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"human_readable": map[string]interface{}{"type": "boolean", "description": "SIZE like lsblk (60G); default is bytes (lsblk -b)"},
						"output_format":  map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
						"all":            map[string]interface{}{"type": "boolean", "description": "Include empty devices and RAM disks (lsblk -a)"},
						"privileged":     map[string]interface{}{"type": "boolean", "description": "Run as root - in containerized deployments, reads the host's mount table for MOUNTPOINTS"},
					},
				},
			},
			map[string]interface{}{
				"name":          "disks/mounts",
				"tools_group":   "disks",
				"linuxctl_verb": "mounts",
				"description":   mountsDesc,
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
						"fs_type":       map[string]interface{}{"type": "string", "description": "Exact filesystem type, no globs (e.g. 'ext4', 'overlay', 'tmpfs')"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused"},
					},
				},
			},
			map[string]interface{}{
				"name":          "disks/performance",
				"tools_group":   "disks",
				"linuxctl_verb": "performance",
				"description":   "Returns block-device I/O counters from /proc/diskstats: reads and writes completed and merged, sectors (512 bytes) and milliseconds spent, in-flight I/Os and weighted I/O time. Values are CUMULATIVE since boot, not rates and not iostat's per-interval figures; there is no %util or await, so sample twice and subtract to get a rate. Read-only. Without `device` all devices are listed except `loop*` and `ram*`; a name such as `sda` (find them with `disks/list`) selects one, and an unknown name gives `no such block device`. Text output is a table; `output_format: json` and `yaml` (real YAML here) return an array of objects with 14 fields (major, minor, device_name, reads_completed, ..., weighted_time_ios_ms). For capacity use `disks/free`, for SMART health `disks/health`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "json or yaml (real YAML) for structured output; default is a text table"},
						"device":        map[string]interface{}{"type": "string", "description": "Block device name such as 'sda' (see disks/list); omit for all except loop* and ram*"},
					},
				},
			},
			map[string]interface{}{
				"name":          "disks/health",
				"tools_group":   "disks",
				"linuxctl_verb": "health",
				"description":   "Returns a drive's SMART data as smartctl's JSON (wraps `smartctl -j -a`; the smartmontools package must be installed or the call fails saying so). Read-only, but normally needs root: use `privileged: true` (needs a grant), since without it smartctl usually cannot open the device. `device` is a bare kernel name such as `sda` or `nvme0n1`, never a path; find names with `disks/list`. The reply is smartctl's JSON verbatim (keys such as smart_status, temperature, ata_smart_attributes or nvme_smart_health_information_log vary by drive type); a non-zero smartctl exit is not an error when it printed JSON, and virtual disks report SMART as unsupported. For I/O counters use `disks/performance`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"device":     map[string]interface{}{"type": "string", "description": "Bare kernel device name such as 'sda' or 'nvme0n1' (never a path)"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root - required to read SMART data. Needs a grant for this tool in mcp-sudo.yaml"},
					},
					"required": []string{"device"},
				},
			},
			map[string]interface{}{
				"name":          "disks/partitions",
				"tools_group":   "disks",
				"linuxctl_verb": "partitions",
				"description":   "Lists the partitions of a disk with start sector and size in sectors and bytes, read natively from /sys/class/block (no fdisk). Read-only. `device` names the PARENT DISK (`sda`, `nvme0n1`), not a partition; without it every disk's partitions are listed. A sector size of 512 bytes is assumed. It does not report partition type, label, UUID or filesystem: use `disks/list` or `disks/mounts` for those, and `disks/list` to find device names. A named disk without partitions is an error (`no partitions found for device`). Output is a text table; when `output_format` is set to any non-empty value (the parameter is accepted although not listed in the schema) it is an indented JSON array of objects (device, parent_disk, number, start_sector, size_sectors, size_bytes).",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"device": map[string]interface{}{"type": "string", "description": "Parent disk name such as 'sda' or 'nvme0n1' (not a partition); omit for all disks"},
					},
				},
			},
			map[string]interface{}{
				"name":          "network/trace-path",
				"tools_group":   "network",
				"linuxctl_verb": "trace-path",
				"description":   "Traces the network path to a host by running the external `traceroute` binary (not tracepath; it must be installed on the host). Read-only, but it sends probe packets. `host` is a hostname or IP; `max_hops` defaults to traceroute's 30 and is capped at 255. There is no timeout parameter and the 30 s worker limit kills slow traces (30 hops x 3 probes can take minutes), so set a low `max_hops` such as 15. Non-responding hops show as `* * *`. Returns traceroute's raw text; there is no `output_format`. Use `network/ping` first for basic reachability, `network/nslookup` for DNS problems.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"host":     map[string]interface{}{"type": "string", "description": "Target hostname or IP address"},
						"max_hops": map[string]interface{}{"type": "integer", "description": "Maximum hops (default 30, capped at 255); keep it low, the 30 s worker limit kills slow traces"},
					},
					"required": []string{"host"},
				},
			},
			map[string]interface{}{
				"name":          "system/os-release",
				"tools_group":   "system",
				"linuxctl_verb": "os-release",
				"description":   "Returns the Linux distribution and kernel version: the contents of /etc/os-release plus the uname line (system, host name, release, version, machine). Read-only. In a containerized daemon /etc/os-release is the container image's, not the host's. Text has an `OS Release Info:` block with the raw file and a `Kernel Info:` line; `output_format: json` (also yaml/table/wide) returns an object with `os_release` (the raw file text, not parsed into fields) and `kernel` (the uname line). For CPU details use `cpu/list`, for kernel parameters `kernel/system-control`.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"}},
				},
			},
			map[string]interface{}{
				"name":          "system/packages",
				"tools_group":   "system",
				"linuxctl_verb": "packages",
				"description":   pkgDesc,
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
						"name":          map[string]interface{}{"type": "string", "description": "Only packages whose name matches this glob or exact name (e.g. 'openssh-*', '*ssl*', 'curl')"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused"},
					},
				},
			},
			map[string]interface{}{
				"name":          "users/list",
				"tools_group":   "users",
				"linuxctl_verb": "get",
				"description":   usersDesc,
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
						"min_uid":       map[string]interface{}{"type": "integer", "description": "Only include users with UID >= this value (e.g. 1000 to exclude system accounts)"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused"},
					},
				},
			},
			map[string]interface{}{
				"name":          "auth/sudo-rules",
				"tools_group":   "auth",
				"linuxctl_verb": "sudo-rules",
				"description":   "Shows which tools YOU may run as root: your `privileged` grants from mcp-sudo.yaml with their `paths`, `containers`, `prune`, `network` and `sysctl` restrictions. It does not list which tools you may call at all: unprivileged calls need no grant, except docker/* and daemon/reload-config, which always do. Read-only and answered by the daemon itself. Call it before a `privileged: true` request or after a permission-denied error. Text is `Your authorized privileged tools:` followed by JSON; `output_format: json` returns only that JSON, whose keys are capitalised Go field names (Tools, Resources, Allowed, Paths, Containers, Prune, Network, Sysctl). No grants gives `You have no privileged tools authorized in mcp-sudo.yaml.` (JSON: `{}`). After an operator edits grants, `daemon/reload-config` applies them.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"}},
				},
			},
		},
	}
	// Every tool is listed for every user; a call the user's grant does not
	// allow is refused with an error naming the missing grant (FR-020).
	toolsList["tools"] = append(toolsList["tools"].([]interface{}), dockerTools()...)
	toolsList["tools"] = append(toolsList["tools"].([]interface{}), map[string]interface{}{
		"name":          "daemon/reload-config",
		"tools_group":   "daemon",
		"linuxctl_verb": "reload",
		"description":   "Re-reads mcpd's config files (daemon.yaml, users.yaml, mcp-sudo.yaml) and applies them without a restart: users and tokens, per-user grants, rate limits and tool timeouts. The files themselves are edited on the host (linuxctl); this only reloads them. Mutating (replaces the in-memory config) and always needs `allowed: true` for `daemon/reload-config` in the caller's grant; there is no unprivileged mode. The files are validated strictly first (a misspelled key is an error): if any is invalid nothing changes and the error is returned. Sessions of removed users and of users whose token changed are closed, possibly the caller's own. Server settings (port, TLS, `worker.containerized`, Docker socket) still need a restart. Returns free text listing what changed. Takes no parameters; verify grants afterwards with `auth/sudo-rules`.",
		"inputSchema": map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	})

	annotateTools(toolsList["tools"].([]interface{}))
	resp.Result = toolsList

}

// redactArgs returns a JSON representation of tool call arguments with
// sensitive-looking fields masked, so the [TOOL CALL] log line can't leak
// secrets or raw file content (e.g. files/create content, network/curl
// Authorization headers) into the daemon's plaintext logs.
func redactArgs(raw json.RawMessage) string {
	var args interface{}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "<unparseable arguments>"
	}
	redacted, err := json.Marshal(redactValue(args))
	if err != nil {
		return "<unparseable arguments>"
	}
	return string(redacted)
}

// redactValue masks sensitive-looking keys at any depth - network/curl's
// "headers": {"Authorization": ...} is nested one level down, which a
// top-level-only check missed.
func redactValue(v interface{}) interface{} {
	sensitiveSubstrings := []string{"password", "token", "secret", "authorization", "cookie", "content", "body", "data", "header", "value", "key"}
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			lk := strings.ToLower(k)
			masked := false
			for _, s := range sensitiveSubstrings {
				if strings.Contains(lk, s) {
					t[k] = "<redacted>"
					masked = true
					break
				}
			}
			if !masked {
				t[k] = redactValue(val)
			}
		}
		return t
	case []interface{}:
		for i := range t {
			t[i] = redactValue(t[i])
		}
		return t
	}
	return v
}

func (h *RPCHandler) HandleToolsCall(session *Session, req JSONRPCRequest, resp *JSONRPCResponse) {
	// One snapshot per request: a concurrent reload can't change the rules
	// between the authorization check and the call it authorizes.
	sudoCfg := h.Sudo()
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err == nil {

		start := time.Now()
		loggedArgs := redactArgs(params.Arguments) // before the path/policy rewrites below

		toolRes := ToolResult{}
		var resultText string
		var execErr error
		defer func() {
			logToolCall(session, params, loggedArgs, start, execErr, resp.Error != nil, resultText)
		}()

		standardWorkers := map[string]bool{
			"files/list":            true,
			"files/read":            true,
			"files/create":          true,
			"files/update":          true,
			"files/find":            true,
			"files/filetype":        true,
			"files/chmod":           true,
			"files/chown":           true,
			"services/manage":       true,
			"services/list":         true,
			"timers/list":           true,
			"logs/journal-control":  true,
			"logs/dmesg":            true,
			"logs/logins":           true,
			"kernel/system-control": true,
			"disks/free":            true,
			"disks/usage":           true,
			"disks/list":            true,
			"disks/mounts":          true,
			"disks/performance":     true,
			"disks/health":          true,
			"disks/partitions":      true,
			"processes/list":        true,
			"processes/top":         true,
			"processes/delete":      true,
			"network/connections":   true,
			"network/nslookup":      true,
			"network/curl":          true,
			"network/arp":           true,
			"network/ping":          true,
			"network/trace-path":    true,
			"memory/usage":          true,
			"cpu/list":              true,
			"cpu/load-average":      true,
			"system/os-release":     true,
			"system/packages":       true,
			"users/list":            true,
			"docker/containers":     true,
			"docker/manage":         true,
			"docker/logs":           true,
			"docker/exec":           true,
			"docker/images":         true,
			"docker/volumes":        true,
			"docker/networks":       true,
			"docker/prune":          true,
		}

		if params.Name == "daemon/reload-config" {
			// Runs in the master: it only re-reads mcpd's own config files.
			if !sudoCfg.CanRunAsRoot(session.User, params.Name) {
				execErr = fmt.Errorf("user %s is not authorized to run %s (grant it in mcp-sudo.yaml)", session.User, params.Name)
			} else if h.ReloadConfig == nil {
				execErr = fmt.Errorf("config reload is not available")
			} else {
				resultText, execErr = h.ReloadConfig(session.User)
			}

		} else if params.Name == "auth/sudo-rules" {
			// No need to spawn an isolated worker to read our own memory config
			resultText, execErr = sudorules.SudoRules(params.Arguments, session.User, sudoCfg)

		} else if standardWorkers[params.Name] {

			// Standard privileged check payload
			var baseArgs struct {
				Privileged bool   `json:"privileged"`
				Path       string `json:"path"`
			}
			_ = json.Unmarshal(params.Arguments, &baseArgs)

			// Docker tools are root or nothing, and the daemon - not the
			// caller - supplies the socket path and the containers allowlist.
			if config.DockerTools[params.Name] {
				baseArgs.Privileged = true
				params.Arguments, execErr = prepareDockerCall(sudoCfg, session.User, params.Name, params.Arguments)
			}

			// Paths are absolute: a relative one would be resolved against
			// the worker's working directory, which no one intends.
			if (config.PathTools[params.Name] || params.Name == "files/stat") && baseArgs.Path != "" && !strings.HasPrefix(baseArgs.Path, "/") {
				execErr = fmt.Errorf("path must be absolute, got %q", baseArgs.Path)
			}

			// Over stdio root is never available: say so before the path
			// check below answers as if a grant were merely missing.
			if execErr == nil && baseArgs.Privileged && worker.NoRoot {
				execErr = worker.ErrNoRoot
			}

			// _no_follow is the daemon's to set, never the caller's.
			params.Arguments = withoutKey(params.Arguments, "_no_follow")

			// Tools with a path argument, run as root, are limited to their
			// grant's paths - no paths, no root (see config.PathTools).
			allowedPaths := sudoCfg.GetAllowedPaths(session.User, params.Name)
			if execErr == nil && config.PathTools[params.Name] && baseArgs.Privileged {
				checkPath := baseArgs.Path
				if checkPath == "" && params.Name == "files/find" {
					checkPath = "/" // files/find's own default
				}
				cleanPath, allowed := config.PathAllowed(checkPath, allowedPaths)
				if !allowed {
					execErr = fmt.Errorf("user %s is not authorized to run %s on path %s as root", session.User, params.Name, baseArgs.Path)
				} else {
					// Hand the worker exactly the path that was authorized,
					// not the raw one - so no later resolution step can
					// reinterpret it differently from the check.
					var argMap map[string]interface{}
					if err := json.Unmarshal(params.Arguments, &argMap); err == nil {
						argMap["path"] = cleanPath
						// A lexical path check says nothing about where a
						// symlink under an allowed directory leads. Unless the
						// grant covers "/" anyway, the worker must not follow
						// symlinks in any component of the path.
						if !config.CoversRoot(allowedPaths) {
							argMap["_no_follow"] = true
						}
						if b, err := json.Marshal(argMap); err == nil {
							params.Arguments = b
						}
					}
				}
			}

			// kernel/system-control writes are checked here, in the master,
			// against the user's sysctl policy - the worker never runs for a
			// refused write.
			if params.Name == "kernel/system-control" && execErr == nil {
				// Decoded with the worker's own parser, and a decode error
				// refuses the call: a lenient decode that dropped a field
				// it couldn't parse (e.g. a numeric value) would skip the
				// policy check while the worker still performed the write.
				sc, err := systemcontrol.ParseArgs(params.Arguments)
				if err != nil {
					execErr = err
				} else if sc.Key != "" && sc.Value != "" {
					if ok, reason := sudoCfg.CanWriteSysctl(session.User, systemcontrol.NormalizeKey(sc.Key)); !ok {
						execErr = fmt.Errorf("%s", reason)
					}
				}
			}

			// Outbound network tools get this user's per-tool network
			// policy injected by the daemon - always set or removed here,
			// so a caller can never supply their own "_network_policy".
			if params.Name == "network/curl" || params.Name == "network/ping" {
				var argMap map[string]interface{}
				if err := json.Unmarshal(params.Arguments, &argMap); err == nil {
					delete(argMap, "_network_policy")
					if pol := sudoCfg.NetworkPolicy(session.User, params.Name); pol != nil {
						argMap["_network_policy"] = pol
					}
					if b, err := json.Marshal(argMap); err == nil {
						params.Arguments = b
					}
				}
			}

			// Resolve execution timeout (check tool override, fallback to global worker default)
			executionTimeout := h.timeoutFor(params.Name)

			if execErr == nil {
				// Use the Ephemeral Worker Spawner with caching for heavy tools
				if params.Name == "disks/usage" {
					cacheKey := fmt.Sprintf("%s:%s:%t", session.User, string(params.Arguments), baseArgs.Privileged)

					h.CacheMu.RLock()
					entry, ok := h.RpcCache[cacheKey]
					h.CacheMu.RUnlock()

					if ok && time.Now().Before(entry.ExpiresAt) {
						resultText = entry.Result
					} else {
						v, err, _ := h.RequestGroup.Do(cacheKey, func() (interface{}, error) {
							res, exErr := worker.SpawnWorker(session.User, params.Name, params.Arguments, baseArgs.Privileged, sudoCfg, executionTimeout)
							if exErr == nil {
								h.CacheMu.Lock()
								h.RpcCache[cacheKey] = CacheEntry{
									Result:    res,
									ExpiresAt: time.Now().Add(60 * time.Second),
								}
								h.CacheMu.Unlock()
							}
							return res, exErr
						})

						if err != nil {
							execErr = err
						} else {
							resultText = v.(string)
						}
					}
				} else {
					resultText, execErr = worker.SpawnWorker(session.User, params.Name, params.Arguments, baseArgs.Privileged, sudoCfg, executionTimeout)
				}
			}

		} else {
			resp.Error = map[string]interface{}{"code": -32601, "message": "Tool not found"}
		}

		if resp.Error == nil {
			if execErr != nil {
				toolRes.IsError = true
				toolRes.Content = append(toolRes.Content, struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{Type: "text", Text: execErr.Error()})
			} else {
				toolRes.Content = append(toolRes.Content, struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{Type: "text", Text: resultText})
			}
			resp.Result = toolRes
		}

	} else {
		resp.Error = map[string]interface{}{"code": -32602, "message": "Invalid params"}
	}
}

// mutatingTools change the host (or, for kernel/system-control, do when
// given a value). Their calls are audit-logged whatever the log level.
// daemon/reload-config audits itself, with the list of changes.
var mutatingTools = map[string]bool{
	"files/create": true, "files/update": true, "files/chmod": true, "files/chown": true,
	"processes/delete": true, "services/manage": true,
	// docker/exec runs arbitrary code inside a container, so it is audited
	// whether or not the command it ran changed anything.
	"docker/manage": true, "docker/exec": true,
	// docker/prune deletes objects nobody named, so it is audited with the
	// target and - uniquely - a one-line summary of what it reclaimed.
	"docker/prune": true,
}

// logToolCall writes one line per tool call: an audit line for calls that
// change something, otherwise an info line (warn when the call was denied).
// Arguments are redacted; tool output is never logged.
func logToolCall(session *Session, params CallToolParams, loggedArgs string, start time.Time, execErr error, rpcErr bool, resultText string) {
	var a struct {
		Privileged bool            `json:"privileged"`
		Value      json.RawMessage `json:"value"`
	}
	_ = json.Unmarshal(params.Arguments, &a)
	attrs := []any{"user", session.User, "session", session.ID, "tool", params.Name, "privileged", a.Privileged,
		"duration_ms", time.Since(start).Milliseconds(), "ok", execErr == nil && !rpcErr, "args", loggedArgs}
	// The one exception to "tool output is never logged": what a prune
	// deleted is the whole point of auditing it, and the summary line names
	// no content - only counts and bytes.
	if params.Name == "docker/prune" && execErr == nil {
		attrs = append(attrs, "reclaimed", pruneSummary(resultText))
	}
	if execErr != nil {
		msg := execErr.Error()
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		attrs = append(attrs, "error", msg)
	}
	mutating := mutatingTools[params.Name] ||
		(params.Name == "kernel/system-control" && len(a.Value) > 0 && string(a.Value) != "null")
	switch {
	case mutating:
		logging.Audit("tool call", attrs...)
	case execErr != nil && strings.Contains(execErr.Error(), "not authorized"):
		logging.Warn("tool call denied", attrs...)
	default:
		logging.Info("tool call", attrs...)
	}
}

// pruneSummary reduces docker/prune's answer to the one line worth auditing:
// the kind reclaimed, how many objects went, and how many bytes came back.
func pruneSummary(resultText string) string {
	// Any structured output_format answers as one JSON line that holds every
	// deleted object's ID - summarize it from the counts, or the "no content"
	// promise above holds only for the text format.
	if strings.HasPrefix(strings.TrimSpace(resultText), "{") {
		var r struct {
			Target    string `json:"target"`
			Count     int    `json:"count"`
			Reclaimed int64  `json:"space_reclaimed_bytes"`
		}
		if err := json.Unmarshal([]byte(resultText), &r); err == nil {
			return fmt.Sprintf("%s: %d removed, %d bytes", r.Target, r.Count, r.Reclaimed)
		}
	}
	// The text format is one header line followed by the deleted objects'
	// own names, indented - the header alone is the summary.
	for _, line := range strings.Split(resultText, "\n") {
		if line == "" || strings.HasPrefix(line, " ") {
			continue
		}
		return strings.TrimSpace(line)
	}
	return ""
}

// withoutKey returns the JSON object args without key (args unchanged if
// it isn't an object or lacks key).
func withoutKey(args json.RawMessage, key string) json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(args, &m); err != nil {
		return args
	}
	if _, ok := m[key]; !ok {
		return args
	}
	delete(m, key)
	b, err := json.Marshal(m)
	if err != nil {
		return args
	}
	return b
}
