package rpc

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
	sudorules "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/auth/sudo-rules"
	systemcontrol "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/kernel/system-control"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/worker"
)

func (h *RPCHandler) HandleToolsList(session *Session, resp *JSONRPCResponse) {
	// Dynamically generate the tools list based on sudo rules.
	listDesc := "Lists a directory like ls -la: file type and permissions, link count, owner, group, size, modification time and symlink targets (symlinks are shown, never followed)."
	if h.SudoConfig.CanRunAsRoot(session.User, "files/list") {
		listDesc += " (Hint: You are authorized to run this tool as root. Use 'privileged: true' if you receive permission denied errors on sensitive paths)."
	}

	dfDesc := "Returns disk space statistics (df -h). Use disks/list to see all block devices."
	if h.SudoConfig.CanRunAsRoot(session.User, "disks/free") {
		dfDesc += " (Authorized for 'privileged: true')"
	}

	duDesc := "Calculates the total disk space utilized by a specific directory (du -sh). Use disks/free for overall partition stats."
	if h.SudoConfig.CanRunAsRoot(session.User, "disks/usage") {
		duDesc += " (Authorized for 'privileged: true' to traverse protected subdirectories)"
	}

	filetypeDesc := "Determines a file's MIME type (equivalent to `file -b --mime-type`). Use files/stat for size/permissions/ownership instead."
	if h.SudoConfig.CanRunAsRoot(session.User, "files/filetype") {
		filetypeDesc += " (Hint: You are authorized to run this tool as root. Use 'privileged: true' if you receive permission denied errors on sensitive paths)."
	}

	pkgDesc := "Lists installed packages, auto-detecting the package manager (dpkg, apk; rpm-based systems aren't supported natively yet)."
	if h.SudoConfig.CanRunAsRoot(session.User, "system/packages") {
		pkgDesc += " (Authorized for 'privileged: true' - when this daemon runs containerized, that automatically queries the real host's packages, not this container's own image.)"
	}

	mountsDesc := "Lists mounted filesystems (device, mount point, type, options) - equivalent to `mount`/`findmnt`'s basic view. Use disks/list for block devices instead."
	if h.SudoConfig.CanRunAsRoot(session.User, "disks/mounts") {
		mountsDesc += " (Authorized for 'privileged: true' - when this daemon runs containerized, that automatically shows the real host's mount table, not this container's own.)"
	}

	usersDesc := "Lists user accounts from /etc/passwd (uid, gid, home, shell, group memberships). Never reads /etc/shadow - this reports account identity, not credentials."
	if h.SudoConfig.CanRunAsRoot(session.User, "users/list") {
		usersDesc += " (Authorized for 'privileged: true' - when this daemon runs containerized, that automatically lists the real host's users, not this container's own.)"
	}

	loginsDesc := "Lists login history (wraps `last`) or failed login attempts (`type: \"failed\"`, wraps `lastb`). Returns raw text, not JSON - last/lastb's output isn't safe to hand-parse into structured data reliably."
	if h.SudoConfig.CanRunAsRoot(session.User, "logs/logins") {
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
						"output_format":  map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"path":           map[string]interface{}{"type": "string", "description": "Directory path to list"},
						"all":            map[string]interface{}{"type": "boolean", "description": "Include dotfiles, . and .. (ls -a)"},
						"long":           map[string]interface{}{"type": "boolean", "description": "Long listing like ls -l: type+permissions, links, owner, group, size, date, symlink target. Default true; false lists names only"},
						"human_readable": map[string]interface{}{"type": "boolean", "description": "Sizes like 4.0K, 1.5M (ls -h)"},
						"sort":           map[string]interface{}{"type": "string", "enum": []string{"name", "size", "time"}, "description": "Sort by name (default), size (largest first) or time (newest first)"},
						"reverse":        map[string]interface{}{"type": "boolean", "description": "Reverse the sort order (ls -r)"},
						"dirs_first":     map[string]interface{}{"type": "boolean", "description": "List directories before files"},
						"numeric_ids":    map[string]interface{}{"type": "boolean", "description": "Show numeric uid/gid instead of names (ls -n)"},
						"privileged":     map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
					},
					"required": []string{"path"},
				},
			},
			map[string]interface{}{
				"name":          "files/read",
				"tools_group":   "files",
				"linuxctl_verb": "get",
				"description":   "Precision reading of file contents with chunking/streaming support.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":       map[string]interface{}{"type": "string", "description": "Path to the file to read"},
						"start_line": map[string]interface{}{"type": "integer", "description": "Starting line number (1-indexed). Takes precedence over byte offsets."},
						"end_line":   map[string]interface{}{"type": "integer", "description": "Ending line number (inclusive)."},
						"offset":     map[string]interface{}{"type": "integer", "description": "Starting byte offset."},
						"limit":      map[string]interface{}{"type": "integer", "description": "Number of bytes to read."},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Set to true to read as root"},
					},
					"required": []string{"path"},
				},
			},
			map[string]interface{}{
				"name":          "files/create",
				"tools_group":   "files",
				"linuxctl_verb": "create",
				"description":   "Create a new file or replace file contents.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":       map[string]interface{}{"type": "string", "description": "Path to the file to create"},
						"content":    map[string]interface{}{"type": "string", "description": "Text content to write to the file"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Set to true to write as root"},
					},
					"required": []string{"path"},
				},
			},
			map[string]interface{}{
				"name":          "files/update",
				"tools_group":   "files",
				"linuxctl_verb": "update",
				"description":   "Programmatically edit a file by appending text or replacing specific line ranges.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":       map[string]interface{}{"type": "string", "description": "Path to the file to edit"},
						"content":    map[string]interface{}{"type": "string", "description": "Text content to insert or append"},
						"append":     map[string]interface{}{"type": "boolean", "description": "If true, appends the content to the end of the file"},
						"start_line": map[string]interface{}{"type": "integer", "description": "Start of the line range to replace (1-indexed)"},
						"end_line":   map[string]interface{}{"type": "integer", "description": "End of the line range to replace (inclusive)"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Set to true to edit as root"},
					},
					"required": []string{"path", "content"},
				},
			},
			map[string]interface{}{
				"name":          "files/find",
				"tools_group":   "files",
				"linuxctl_verb": "find",
				"description":   "Search for files in a directory hierarchy.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format. Defaults to text"},
						"path":          map[string]interface{}{"type": "string", "description": "Starting directory for the search. Defaults to '/'"},
						"name":          map[string]interface{}{"type": "string", "description": "Glob pattern to match filenames"},
						"type":          map[string]interface{}{"type": "string", "description": "File type ('f' for file, 'd' for directory, 'l' for symlink)"},
						"mtime":         map[string]interface{}{"type": "string", "description": "Modification time (e.g. '+7' for older than 7 days)"},
						"size":          map[string]interface{}{"type": "string", "description": "File size (e.g. '+100M' for larger than 100MB)"},
						"max_depth":     map[string]interface{}{"type": "integer", "description": "Maximum depth for directory recursion"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Set to true to search as root"},
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
						"privileged": map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
					},
					"required": []string{"path"},
				},
			},
			map[string]interface{}{
				"name":          "files/chmod",
				"tools_group":   "files",
				"linuxctl_verb": "chmod",
				"description":   "Changes a file's or directory's permission bits (chmod). Never follows symbolic links: a path containing a symlink in any component is refused, and recursive changes skip symlinks and report them. Numeric modes follow GNU chmod semantics (on directories a 4-digit mode keeps setuid/setgid; use 5 digits, e.g. 00755, to set them exactly).",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":       map[string]interface{}{"type": "string", "description": "Absolute path"},
						"mode":       map[string]interface{}{"type": "string", "description": "Octal (644, 0755, 4755) or symbolic (u+x, go-w, a=r, +X, u+s, +t; comma-separated)"},
						"recursive":  map[string]interface{}{"type": "boolean", "description": "Also apply to everything below a directory (symlinks are skipped, never followed)"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root - needed for files you don't own"},
					},
					"required": []string{"path", "mode"},
				},
			},
			map[string]interface{}{
				"name":          "files/chown",
				"tools_group":   "files",
				"linuxctl_verb": "chown",
				"description":   "Changes a file's or directory's owner and/or group (chown). Never follows symbolic links: a path containing a symlink in any component is refused, and recursive changes skip symlinks and report them. Changing the owner requires privileged: true.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":       map[string]interface{}{"type": "string", "description": "Absolute path"},
						"owner":      map[string]interface{}{"type": "string", "description": "user, user:group, :group, or user: (the user's login group); names or numeric ids"},
						"recursive":  map[string]interface{}{"type": "boolean", "description": "Also apply to everything below a directory (symlinks are skipped, never followed)"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root - required to change ownership"},
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
						"output_format":  map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"path":           map[string]interface{}{"type": "string", "description": "Absolute path to check"},
						"inodes":         map[string]interface{}{"type": "boolean", "description": "List inode information instead of block usage (-i)"},
						"human_readable": map[string]interface{}{"type": "boolean", "description": "Print sizes in powers of 1024 (-h)"},
						"privileged":     map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
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
						"output_format":   map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"path":            map[string]interface{}{"type": "string", "description": "Target directory to measure"},
						"max_depth":       map[string]interface{}{"type": "integer", "description": "How deep to recurse (0 for summarize only)"},
						"one_file_system": map[string]interface{}{"type": "boolean", "description": "Skip directories on different file systems (-x)"},
						"exclude":         map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Patterns to exclude"},
						"all":             map[string]interface{}{"type": "boolean", "description": "Write counts for all files, not just directories (-a)"},
						"apparent_size":   map[string]interface{}{"type": "boolean", "description": "Print apparent sizes rather than device usage (--apparent-size)"},
						"threshold":       map[string]interface{}{"type": "integer", "description": "Exclude entries smaller than SIZE if positive, or greater than SIZE if negative (-t)"},
						"separate_dirs":   map[string]interface{}{"type": "boolean", "description": "For directories do not include size of subdirectories (-S)"},
						"privileged":      map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
					},
					"required": []string{"path"},
				},
			},
			map[string]interface{}{
				"name":          "processes/top",
				"tools_group":   "processes",
				"linuxctl_verb": "top",
				"description":   "A snapshot like `top -b -n 1`: header with uptime, logged-in users, load average, task counts by state, CPU breakdown (us/sy/ni/id/wa/hi/si/st) and memory/swap in MiB, followed by the process table with all of top's columns (PID USER PR NI VIRT RES SHR S %CPU %MEM TIME+ COMMAND). %CPU is measured over a short sampling interval, as top does. Use processes/list for a plain listing, processes/delete to signal a process.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"sort_by":       map[string]interface{}{"type": "string", "enum": []string{"cpu", "mem", "res", "time", "pid"}, "description": "Sort column: cpu (default, like top), mem/res (resident memory), time (total CPU time), pid"},
						"limit":         map[string]interface{}{"type": "integer", "description": "Maximum processes to list (default: all)"},
						"user":          map[string]interface{}{"type": "string", "description": "Only this user's processes"},
						"interval_ms":   map[string]interface{}{"type": "integer", "description": "%CPU sampling interval in milliseconds (default 1000, max 10000)"},
						"output_format": map[string]interface{}{"type": "string", "description": "Default/table: top's own layout. wide: adds PPID, THR and full command lines (like top -c). json/yaml: structured {summary, processes}"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
					},
				},
			},
			map[string]interface{}{
				"name":          "processes/list",
				"tools_group":   "processes",
				"linuxctl_verb": "get",
				"description":   "Lists running processes on the system. Use this to find a PID, then use the process://{pid}/{target} resource for deep metrics or processes/delete to kill it.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"user":          map[string]interface{}{"type": "string", "description": "Filter by username"},
						"pid":           map[string]interface{}{"type": "integer", "description": "Filter to a single specific PID"},
						"sort_by":       map[string]interface{}{"type": "string", "description": "Sort by cpu, mem, or pid"},
						"limit":         map[string]interface{}{"type": "integer", "description": "Limit returned processes"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
					},
				},
			},
			map[string]interface{}{
				"name":          "processes/delete",
				"tools_group":   "processes",
				"linuxctl_verb": "delete",
				"description":   "Terminates a specific process by PID.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"pid":           map[string]interface{}{"type": "integer", "description": "The PID to kill"},
						"signal":        map[string]interface{}{"type": "string", "description": "Signal to send (e.g., SIGTERM, SIGKILL)"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root to kill other user's processes"},
					},
					"required": []string{"pid"},
				},
			},
			map[string]interface{}{
				"name":          "network/nslookup",
				"tools_group":   "network",
				"linuxctl_verb": "nslookup",
				"description":   "Query DNS records natively.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"host":        map[string]interface{}{"type": "string"},
						"record_type": map[string]interface{}{"type": "string", "description": "e.g. A, TXT, MX, CNAME, NS, or ANY"},
					},
					"required": []string{"host"},
				},
			},
			map[string]interface{}{
				"name":          "network/curl",
				"tools_group":   "network",
				"linuxctl_verb": "curl",
				"description":   "Transfer data from a URL using native HTTP client.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"url":      map[string]interface{}{"type": "string"},
						"method":   map[string]interface{}{"type": "string"},
						"body":     map[string]interface{}{"type": "string", "description": "Request body"},
						"headers":  map[string]interface{}{"type": "object", "description": "Request headers, e.g. {\"Content-Type\": \"application/json\"}", "additionalProperties": map[string]interface{}{"type": "string"}},
						"insecure": map[string]interface{}{"type": "boolean", "description": "Skip TLS certificate verification"},
						"timeout":  map[string]interface{}{"type": "number"},
					},
					"required": []string{"url"},
				},
			},
			map[string]interface{}{
				"name":          "network/arp",
				"tools_group":   "network",
				"linuxctl_verb": "arp",
				"description":   "View the system ARP cache (IP to MAC address mappings).",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"interface": map[string]interface{}{"type": "string"},
					},
				},
			},
			map[string]interface{}{
				"name":          "network/ping",
				"tools_group":   "network",
				"linuxctl_verb": "ping",
				"description":   "Measure TCP reachability and latency to a host.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"host":    map[string]interface{}{"type": "string"},
						"port":    map[string]interface{}{"type": "number", "description": "Defaults to 80"},
						"timeout": map[string]interface{}{"type": "number"},
					},
					"required": []string{"host"},
				},
			},
			map[string]interface{}{
				"name":          "network/connections",
				"tools_group":   "network",
				"linuxctl_verb": "connections",
				"description":   "Lists active network connections and listening ports. Hint: For physical network links and IPs, use the network://interfaces resource.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"state":         map[string]interface{}{"type": "string", "description": "Filter by TCP state, case-insensitive (LISTEN/listening, ESTABLISHED, TIME_WAIT, CLOSE_WAIT, SYN_SENT, ...). Omit to list all sockets - active connections and listening ports."},
						"port":          map[string]interface{}{"type": "integer", "description": "Filter by port"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root to see PIDs of other users"},
					},
				},
			},
			map[string]interface{}{
				"name":          "memory/usage",
				"tools_group":   "memory",
				"linuxctl_verb": "usage",
				"description":   "Returns memory and swap utilization information. Use cpu/load-average to check compute load.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"detailed":      map[string]interface{}{"type": "boolean", "description": "Set to true to return raw /proc/meminfo instead of summary"},
					},
				},
			},
			map[string]interface{}{
				"name":          "services/manage",
				"tools_group":   "system",
				"linuxctl_verb": "services",
				"description":   "Control systemd services (start, stop, restart, enable, disable). To get detailed service properties and state, read the service://{name}/status resource. To view service logs, use the logs/journal-control tool.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"service":    map[string]interface{}{"type": "string", "description": "Service name (e.g., 'kubelet.service')"},
						"action":     map[string]interface{}{"type": "string", "enum": []string{"start", "stop", "restart", "reload", "enable", "disable"}, "description": "Action to perform on the service"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root"},
					},
					"required": []string{"service", "action"},
				},
			},
			map[string]interface{}{
				"name":          "services/list",
				"tools_group":   "system",
				"linuxctl_verb": "services",
				"description":   "Lists systemd services with optional filtering. Output includes ActiveState, LoadState, and SubState.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pattern":       map[string]interface{}{"type": "string", "description": "Wildcard pattern to match service names (e.g., 'kube*', '*ssh*')"},
						"active_state":  map[string]interface{}{"type": "string", "description": "Filter by active state (e.g., 'active', 'failed', 'inactive')"},
						"load_state":    map[string]interface{}{"type": "string", "description": "Filter by load state (e.g., 'loaded', 'not-found')"},
						"sub_state":     map[string]interface{}{"type": "string", "description": "Filter by sub state (e.g., 'running', 'exited', 'dead')"},
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, table, wide). Defaults to text"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root (may be required depending on policies)"},
					},
					"required": []string{},
				},
			},
			map[string]interface{}{
				"name":          "logs/journal-control",
				"tools_group":   "logs",
				"linuxctl_verb": "journal",
				"description":   "Queries the systemd journal (journalctl equivalent). Requires privileged: true in containerized deployments, since journalctl only exists on the host, never in this daemon's own image.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"unit":          map[string]interface{}{"type": "string", "description": "Filter by systemd unit (e.g., 'kubelet.service')"},
						"lines":         map[string]interface{}{"type": "integer", "description": "Number of lines to tail (default: 100)"},
						"since":         map[string]interface{}{"type": "string", "description": "Filter logs since a specific time (e.g., '1 hour ago', 'today')"},
						"until":         map[string]interface{}{"type": "string", "description": "Filter logs until a specific time (e.g., 'yesterday', '12:00')"},
						"reverse":       map[string]interface{}{"type": "boolean", "description": "Output newest entries first"},
						"boot":          map[string]interface{}{"type": "boolean", "description": "Restrict output to the current boot (journalctl -b)"},
						"boot_offset":   map[string]interface{}{"type": "integer", "description": "Select a prior boot relative to the current one, e.g. -1 for the previous boot (implies boot)"},
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json). Defaults to text"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root and join the host mount namespace - required in containerized deployments"},
					},
				},
			},
			map[string]interface{}{
				"name":          "logs/dmesg",
				"tools_group":   "logs",
				"linuxctl_verb": "dmesg",
				"description":   "Read the kernel ring buffer for hardware/driver logs.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"level":         map[string]interface{}{"type": "string", "description": "Filter by log level (e.g., 'err,warn')"},
						"output_format": map[string]interface{}{"type": "string", "description": "Output format"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root"},
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
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root - typically required for type: \"failed\""},
					},
				},
			},
			map[string]interface{}{
				"name":          "kernel/system-control",
				"tools_group":   "kernel",
				"linuxctl_verb": "sysctl",
				"description":   "Reads or writes kernel parameters (sysctl equivalent) at runtime, natively via /proc/sys. Writes require privileged: true, and may be restricted per user (read-only, or only certain keys) by mcp-sudo.yaml.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"key":        map[string]interface{}{"type": "string", "description": "Kernel parameter name, dotted (net.ipv4.ip_forward) or slash form (net/ipv4/conf/eth0.100/rp_filter). A directory (e.g. net.ipv4) reads its whole subtree."},
						"value":      map[string]interface{}{"type": "string", "description": "Value to set for the parameter. If omitted, reads the parameter."},
						"read_all":   map[string]interface{}{"type": "boolean", "description": "If true, reads all available parameters. Ignored if key is set."},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root - required for writes"},
					},
				},
			},
			map[string]interface{}{
				"name":          "cpu/list",
				"tools_group":   "cpu",
				"linuxctl_verb": "get",
				"description":   "Retrieves CPU topology and architecture. See cpu/load-average for current utilization.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"topology_only": map[string]interface{}{"type": "boolean", "description": "Only return basic core topology"},
					},
				},
			},
			map[string]interface{}{
				"name":          "cpu/load-average",
				"tools_group":   "cpu",
				"linuxctl_verb": "load-average",
				"description":   "Retrieves system load averages (1m, 5m, 15m). See cpu/list for hardware topology.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"}},
				},
			},
			map[string]interface{}{
				"name":          "disks/list",
				"tools_group":   "disks",
				"linuxctl_verb": "get",
				"description":   "Lists block devices as a tree (equivalent to lsblk): disks, their partitions, and LVM/dm-crypt/RAID volumes nested under the devices they're built on, with MAJ:MIN, RM, SIZE, RO, TYPE and MOUNTPOINTS. json/yaml output is the same tree under \"blockdevices\" (like lsblk -J), with nested \"children\". To check remaining free space or inode usage, use the disks/free tool. To check which folders are taking up the most space, use the disks/usage tool. (Use 'privileged: true' in containerized deployments to see the host's mount points.)",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"all":           map[string]interface{}{"type": "boolean", "description": "Include empty devices and RAM disks (lsblk -a)"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root - in containerized deployments, reads the host's mount table for MOUNTPOINTS"},
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
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"fs_type":       map[string]interface{}{"type": "string", "description": "Only include mounts of this filesystem type (e.g. 'ext4', 'overlay', 'tmpfs')"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root"},
					},
				},
			},
			map[string]interface{}{
				"name":          "disks/performance",
				"tools_group":   "disks",
				"linuxctl_verb": "performance",
				"description":   "Retrieves granular block device I/O performance metrics (equivalent to iostat). Provides read/write sectors, merged operations, and I/O wait times in milliseconds. Use disks/list first to find valid block devices. If you want static capacity instead, use disks/free.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table). Defaults to text"},
						"device":        map[string]interface{}{"type": "string", "description": "Optional specific block device to query (e.g., 'sda')"},
					},
				},
			},
			map[string]interface{}{
				"name":          "disks/health",
				"tools_group":   "disks",
				"linuxctl_verb": "health",
				"description":   "Retrieves detailed SMART health data for a drive (equivalent to smartctl -j -a). Returns JSON containing self-assessment test results, temperature, wear leveling, and sector errors. Must be run as root (privileged: true). Use this to diagnose failing hardware.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"device":     map[string]interface{}{"type": "string", "description": "Specific block device to query (e.g., 'sda')"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root - required to read SMART data"},
					},
					"required": []string{"device"},
				},
			},
			map[string]interface{}{
				"name":          "disks/partitions",
				"tools_group":   "disks",
				"linuxctl_verb": "partitions",
				"description":   "Retrieves partition boundaries for a drive (start/size, in sectors and bytes), parsed natively from /sys/class/block - no fdisk dependency. Use this to understand the low-level geometry and partition boundaries of a disk.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"device": map[string]interface{}{"type": "string", "description": "Optional specific block device to query (e.g., 'sda')"},
					},
				},
			},
			map[string]interface{}{
				"name":          "network/trace-path",
				"tools_group":   "network",
				"linuxctl_verb": "trace-path",
				"description":   "Traces the network path to a host (equivalent to traceroute). Useful for debugging routing issues, identifying where packets are dropped, or measuring network latency across hops. Hint: Use network/ping for basic reachability before tracing the path.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"host":     map[string]interface{}{"type": "string", "description": "Target hostname or IP"},
						"max_hops": map[string]interface{}{"type": "integer", "description": "Maximum number of hops (optional)"},
					},
					"required": []string{"host"},
				},
			},
			map[string]interface{}{
				"name":          "system/os-release",
				"tools_group":   "system",
				"linuxctl_verb": "os-release",
				"description":   "Retrieves Linux distribution and kernel version.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"}},
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
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"name":          map[string]interface{}{"type": "string", "description": "Only packages whose name matches this glob or exact name (e.g. 'openssh-*', '*ssl*', 'curl')"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
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
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"min_uid":       map[string]interface{}{"type": "integer", "description": "Only include users with UID >= this value (e.g. 1000 to exclude system accounts)"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
					},
				},
			},
			map[string]interface{}{
				"name":          "auth/sudo-rules",
				"tools_group":   "auth",
				"linuxctl_verb": "sudo-rules",
				"description":   "Returns your authorized tools and privileges from mcp-sudo.yaml.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"}},
				},
			},
		},
	}
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
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err == nil {

		log.Printf("[TOOL CALL] user=%s tool=%s args=%s", session.User, params.Name, redactArgs(params.Arguments))

		toolRes := ToolResult{}
		var resultText string
		var execErr error

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
		}

		if params.Name == "auth/sudo-rules" {
			// No need to spawn an isolated worker to read our own memory config
			resultText, execErr = sudorules.SudoRules(params.Arguments, session.User, h.SudoConfig)

		} else if standardWorkers[params.Name] {

			// Standard privileged check payload
			var baseArgs struct {
				Privileged bool   `json:"privileged"`
				Path       string `json:"path"`
			}
			_ = json.Unmarshal(params.Arguments, &baseArgs)

			if (params.Name == "files/list" || params.Name == "files/read" || params.Name == "files/create" || params.Name == "files/update" || params.Name == "files/find" || params.Name == "files/filetype" || params.Name == "files/chmod" || params.Name == "files/chown") && baseArgs.Privileged {
				checkPath := baseArgs.Path
				if checkPath == "" && params.Name == "files/find" {
					checkPath = "/" // files/find's own default
				}
				cleanPath, allowed := config.PathAllowed(checkPath, h.SudoConfig.GetAllowedPaths(session.User, params.Name))
				if !allowed {
					execErr = fmt.Errorf("user %s is not authorized to run %s on path %s as root", session.User, params.Name, baseArgs.Path)
				} else {
					// Hand the worker exactly the path that was authorized,
					// not the raw one - so no later resolution step can
					// reinterpret it differently from the check.
					var argMap map[string]interface{}
					if err := json.Unmarshal(params.Arguments, &argMap); err == nil {
						argMap["path"] = cleanPath
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
					if ok, reason := h.SudoConfig.CanWriteSysctl(session.User, systemcontrol.NormalizeKey(sc.Key)); !ok {
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
					if pol := h.SudoConfig.NetworkPolicy(session.User, params.Name); pol != nil {
						argMap["_network_policy"] = pol
					}
					if b, err := json.Marshal(argMap); err == nil {
						params.Arguments = b
					}
				}
			}

			// Resolve execution timeout (check tool override, fallback to global worker default)
			executionTimeout := h.WorkerTimeoutSec
			if toolCfg, ok := h.ToolsConfig[params.Name]; ok && toolCfg.TimeoutSeconds > 0 {
				executionTimeout = toolCfg.TimeoutSeconds
			}

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
							res, exErr := worker.SpawnWorker(session.User, params.Name, params.Arguments, baseArgs.Privileged, h.SudoConfig, executionTimeout)
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
					resultText, execErr = worker.SpawnWorker(session.User, params.Name, params.Arguments, baseArgs.Privileged, h.SudoConfig, executionTimeout)
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
