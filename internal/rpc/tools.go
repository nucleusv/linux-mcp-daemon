package rpc

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sudorules "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/auth/sudo-rules"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/worker"
)

func (h *RPCHandler) HandleToolsList(session *Session, resp *JSONRPCResponse) {
	// Dynamically generate the tools list based on sudo rules.
	listDesc := "Lists contents of a directory."
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

	toolsList := map[string]interface{}{
		"tools": []interface{}{
			map[string]interface{}{
				"name":        "files/list",
				"tools_group": "files",
				"description": listDesc,
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"path":          map[string]interface{}{"type": "string", "description": "Directory path to list"},
						"all":           map[string]interface{}{"type": "boolean", "description": "Include hidden files (-a)"},
						"long":          map[string]interface{}{"type": "boolean", "description": "Use long listing format (-l)"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
					},
					"required": []string{"path"},
				},
			},
			map[string]interface{}{
				"name":        "files/read",
				"tools_group": "files",
				"description": "Precision reading of file contents with chunking/streaming support.",
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
				"name":        "files/create",
				"tools_group": "files",
				"description": "Create a new file or replace file contents.",
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
				"name":        "files/update",
				"tools_group": "files",
				"description": "Programmatically edit a file by appending text or replacing specific line ranges.",
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
				"name":        "files/find",
				"tools_group": "files",
				"description": "Search for files in a directory hierarchy.",
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
				"name":        "disks/free",
				"tools_group": "disks",
				"description": dfDesc,
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
				"name":        "disks/usage",
				"tools_group": "disks",
				"description": duDesc,
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
				"name":        "processes/list",
				"tools_group": "processes",
				"description": "Lists running processes on the system. Use this to find a PID, then use the process://{pid}/{target} resource for deep metrics or processes/delete to kill it.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"user":          map[string]interface{}{"type": "string", "description": "Filter by username"},
						"sort_by":       map[string]interface{}{"type": "string", "description": "Sort by cpu, mem, or pid"},
						"limit":         map[string]interface{}{"type": "integer", "description": "Limit returned processes"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
					},
				},
			},
			map[string]interface{}{
				"name":        "processes/delete",
				"tools_group": "processes",
				"description": "Terminates a specific process by PID.",
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
				"name":        "network/nslookup",
				"tools_group": "network",
				"description": "Query DNS records natively.",
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
				"name":        "network/curl",
				"tools_group": "network",
				"description": "Transfer data from a URL using native HTTP client.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"url":     map[string]interface{}{"type": "string"},
						"method":  map[string]interface{}{"type": "string"},
						"body":    map[string]interface{}{"type": "string"},
						"timeout": map[string]interface{}{"type": "number"},
					},
					"required": []string{"url"},
				},
			},
			map[string]interface{}{
				"name":        "network/arp",
				"tools_group": "network",
				"description": "View the system ARP cache (IP to MAC address mappings).",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"interface": map[string]interface{}{"type": "string"},
					},
				},
			},
			map[string]interface{}{
				"name":        "network/ping",
				"tools_group": "network",
				"description": "Measure TCP reachability and latency to a host.",
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
				"name":        "network/connections",
				"tools_group": "network",
				"description": "Lists active network connections and listening ports. Hint: For physical network links and IPs, use the network://interfaces resource.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"state":         map[string]interface{}{"type": "string", "description": "Filter by TCP state (e.g., LISTEN, ESTABLISHED)"},
						"port":          map[string]interface{}{"type": "integer", "description": "Filter by port"},
						"privileged":    map[string]interface{}{"type": "boolean", "description": "Run as root to see PIDs of other users"},
					},
				},
			},
			map[string]interface{}{
				"name":        "memory/usage",
				"tools_group": "memory",
				"description": "Returns memory and swap utilization information. Use cpu/load-average to check compute load.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"detailed":      map[string]interface{}{"type": "boolean", "description": "Set to true to return raw /proc/meminfo instead of summary"},
					},
				},
			},
			map[string]interface{}{
				"name":        "services/manage",
				"tools_group": "system",
				"description": "Control systemd services (start, stop, restart, enable, disable). To get detailed service properties and state, read the service://{name}/status resource. To view service logs, use the logs/journalctl tool.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"service":    map[string]interface{}{"type": "string", "description": "Service name (e.g., 'kubelet.service')"},
						"action":     map[string]interface{}{"type": "string", "description": "Action (start, stop, restart, reload, enable, disable)"},
						"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root"},
					},
					"required": []string{"service", "action"},
				},
			},
			map[string]interface{}{
				"name":        "services/list",
				"tools_group": "system",
				"description": "Lists systemd services with optional filtering. Output includes ActiveState, LoadState, and SubState.",
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
				"name":        "logs/journal-control",
				"tools_group": "logs",
				"description": "Queries the systemd journal (journalctl equivalent).",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"unit":    map[string]interface{}{"type": "string", "description": "Filter by systemd unit (e.g., 'kubelet.service')"},
						"lines":   map[string]interface{}{"type": "integer", "description": "Number of lines to tail (default: 100)"},
						"since":   map[string]interface{}{"type": "string", "description": "Filter logs since a specific time (e.g., '1 hour ago', 'today')"},
						"until":   map[string]interface{}{"type": "string", "description": "Filter logs until a specific time (e.g., 'yesterday', '12:00')"},
						"reverse": map[string]interface{}{"type": "boolean", "description": "Output newest entries first"},
					},
				},
			},
			map[string]interface{}{
				"name":        "logs/dmesg",
				"tools_group": "logs",
				"description": "Read the kernel ring buffer for hardware/driver logs.",
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
				"name":        "kernel/system-control",
				"tools_group": "kernel",
				"description": "Reads or writes kernel parameters (sysctl equivalent) at runtime.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"key":      map[string]interface{}{"type": "string", "description": "Kernel parameter name (e.g., net.ipv4.ip_forward)"},
						"value":    map[string]interface{}{"type": "string", "description": "Value to set for the parameter. If omitted, reads the parameter."},
						"read_all": map[string]interface{}{"type": "boolean", "description": "If true, reads all available parameters. Ignored if key is set."},
					},
				},
			},
			map[string]interface{}{
				"name":        "cpu/list",
				"tools_group": "cpu",
				"description": "Retrieves CPU topology and architecture. See cpu/load-average for current utilization.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"topology_only": map[string]interface{}{"type": "boolean", "description": "Only return basic core topology"},
					},
				},
			},
			map[string]interface{}{
				"name":        "cpu/load-average",
				"tools_group": "cpu",
				"description": "Retrieves system load averages (1m, 5m, 15m). See cpu/list for hardware topology.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"}},
				},
			},
			map[string]interface{}{
				"name":        "disks/list",
				"tools_group": "disks",
				"description": "Lists block devices and partitions. To check remaining free space or inode usage, use the disks/free tool. To check which folders are taking up the most space, use the disks/usage tool.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"all":           map[string]interface{}{"type": "boolean", "description": "Include empty devices"},
					},
				},
			},
			map[string]interface{}{
				"name":        "disks/performance",
				"tools_group": "disks",
				"description": "Retrieves granular block device I/O performance metrics (equivalent to iostat). Provides read/write sectors, merged operations, and I/O wait times in milliseconds. Use disks/list first to find valid block devices. If you want static capacity instead, use disks/free.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table). Defaults to text"},
						"device":        map[string]interface{}{"type": "string", "description": "Optional specific block device to query (e.g., 'sda')"},
					},
				},
			},
			map[string]interface{}{
				"name":        "disks/health",
				"tools_group": "disks",
				"description": "Retrieves detailed SMART health data for a drive (equivalent to smartctl -j -a). Returns JSON containing self-assessment test results, temperature, wear leveling, and sector errors. Must be run as root (privileged: true). Use this to diagnose failing hardware.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"device": map[string]interface{}{"type": "string", "description": "Specific block device to query (e.g., 'sda')"},
					},
					"required": []string{"device"},
				},
			},
			map[string]interface{}{
				"name":        "disks/partitions",
				"tools_group": "disks",
				"description": "Retrieves detailed partition tables for a drive (equivalent to fdisk -l). Returns raw text. Must be run as root (privileged: true). Use this to understand the low-level geometry and partition boundaries of a disk.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"device": map[string]interface{}{"type": "string", "description": "Optional specific block device to query (e.g., 'sda')"},
					},
				},
			},
			map[string]interface{}{
				"name":        "network/trace-path",
				"tools_group": "network",
				"description": "Traces the network path to a host (equivalent to traceroute). Useful for debugging routing issues, identifying where packets are dropped, or measuring network latency across hops. Hint: Use network/ping for basic reachability before tracing the path.",
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
				"name":        "system/os-release",
				"tools_group": "system",
				"description": "Retrieves Linux distribution and kernel version.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"}},
				},
			},
			map[string]interface{}{
				"name":        "auth/sudo-rules",
				"tools_group": "auth",
				"description": "Returns your authorized tools and privileges from mcp-sudo.yaml.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"}},
				},
			},
			map[string]interface{}{
				"name":        "nslookup",
				"tools_group": "network",
				"description": "Resolves a hostname to an IP address.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"host":          map[string]interface{}{"type": "string", "description": "Hostname to resolve"},
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format"},
					},
					"required": []string{"host"},
				},
			},
			map[string]interface{}{
				"name":        "curl",
				"tools_group": "network",
				"description": "Executes an HTTP GET request.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"url":           map[string]interface{}{"type": "string", "description": "URL to fetch"},
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format"},
					},
					"required": []string{"url"},
				},
			},
			map[string]interface{}{
				"name":        "arp",
				"tools_group": "network",
				"description": "Displays the ARP cache.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format"},
					},
				},
			},
			map[string]interface{}{
				"name":        "ping",
				"tools_group": "network",
				"description": "Sends ICMP echo requests.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"host":          map[string]interface{}{"type": "string", "description": "Host to ping"},
						"count":         map[string]interface{}{"type": "integer", "description": "Number of packets"},
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format"},
					},
					"required": []string{"host"},
				},
			},
		},
	}
	resp.Result = toolsList

}

func (h *RPCHandler) HandleToolsCall(session *Session, req JSONRPCRequest, resp *JSONRPCResponse) {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err == nil {

		toolRes := ToolResult{}
		var resultText string
		var execErr error

		standardWorkers := map[string]bool{
			"files/list":          true,
			"files/read":          true,
			"files/create":        true,
			"files/update":        true,
			"files/find":          true,
			"services/manage":     true,
			"services/list":       true,
			"logs/journal-control":  true,
			"logs/dmesg":          true,
			"kernel/system-control": true,
			"disks/free":          true,
			"disks/usage":         true,
			"disks/list":          true,
			"disks/performance":   true,
			"disks/health":        true,
			"disks/partitions":    true,
			"processes/list":      true,
			"processes/delete":    true,
			"network/connections": true,
			"network/nslookup":    true,
			"network/curl":        true,
			"network/arp":         true,
			"network/ping":        true,
			"network/trace-path":  true,
			"memory/usage":        true,
			"cpu/list":            true,
			"cpu/load-average":    true,
			"system/os-release":   true,
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

			if (params.Name == "files/list" || params.Name == "files/read" || params.Name == "files/create" || params.Name == "files/update" || params.Name == "files/find") && baseArgs.Privileged {
				allowedPaths := h.SudoConfig.GetAllowedPaths(session.User, params.Name)
				allowed := false
				for _, p := range allowedPaths {
					if strings.HasPrefix(baseArgs.Path, p) {
						allowed = true
						break
					}
				}
				if !allowed {
					execErr = fmt.Errorf("user %s is not authorized to run %s on path %s as root", session.User, params.Name, baseArgs.Path)
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
