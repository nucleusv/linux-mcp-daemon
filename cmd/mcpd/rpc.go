package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"syscall"
	"time"

	sudorules "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/auth/sudo-rules"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/worker"
)

func processJSONRPC(session *Session, req JSONRPCRequest) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		handleInitialize(&resp)
	case "notifications/initialized":
		return
	case "ping":
		// Ping is just an empty map, handled implicitly
	case "resources/list":
		handleResourcesList(&resp)
	case "resources/templates/list":
		handleResourcesTemplatesList(&resp)
	case "resources/read":
		handleResourcesRead(session, req, &resp)
	case "tools/list":
		handleToolsList(session, &resp)
	case "tools/call":
		handleToolsCall(session, req, &resp)
	default:
		resp.Error = map[string]interface{}{"code": -32601, "message": "Method not found"}
	}
	respBytes, _ := json.Marshal(resp)
	log.Printf("Sending response for %s: %s", req.Method, string(respBytes))
	session.Event <- string(respBytes)
}

func handleInitialize(resp *JSONRPCResponse) {
	resp.Result = map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools":     map[string]interface{}{},
			"resources": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "linux-mcp-daemon",
			"version": "1.0.0",
		},
	}
}

func handleResourcesList(resp *JSONRPCResponse) {
	// Enumerate static resources and wildcard templates
	resp.Result = map[string]interface{}{
		"resources": []interface{}{
			map[string]interface{}{
				"uri":         "os://uname",
				"name":        "OS Uname",
				"description": "Native system uname information",
				"mimeType":    "text/plain",
			},
			map[string]interface{}{
				"uri":         "os://release",
				"name":        "OS Release",
				"description": "/etc/os-release information",
				"mimeType":    "text/plain",
			},
			map[string]interface{}{
				"uri":         "os://hostname",
				"name":        "OS Hostname",
				"description": "Native system network hostname",
				"mimeType":    "text/plain",
			},
			map[string]interface{}{
				"uri":         "network://interfaces",
				"name":        "Network Interfaces",
				"description": "Network interfaces and assigned IP addresses (ip addr equivalent)",
				"mimeType":    "application/json",
			},
			map[string]interface{}{
				"uri":         "network://routes",
				"name":        "Network Routes",
				"description": "IPv4 Routing Table (/proc/net/route)",
				"mimeType":    "application/json",
			},
			map[string]interface{}{
				"uri":         "devices://usb",
				"name":        "USB Devices",
				"description": "Connected USB devices (lsusb equivalent)",
				"mimeType":    "application/json",
			},
			map[string]interface{}{
				"uri":         "devices://pci",
				"name":        "PCI Devices",
				"description": "Connected PCI devices (lspci equivalent)",
				"mimeType":    "application/json",
			},
			map[string]interface{}{
				"uri":         "devices://dmi",
				"name":        "DMI Hardware Info",
				"description": "Desktop Management Interface info (lshw/hwinfo equivalent)",
				"mimeType":    "application/json",
			},
			map[string]interface{}{
				"uri":         "kernel://modules",
				"name":        "Kernel Modules",
				"description": "Loaded kernel drivers (lsmod equivalent)",
				"mimeType":    "application/json",
			},
		},
	}
}

func handleResourcesTemplatesList(resp *JSONRPCResponse) {
	resp.Result = map[string]interface{}{
		"resourceTemplates": []interface{}{
			map[string]interface{}{
				"uriTemplate": "file:///{path}",
				"name":        "File Reader",
				"description": "Reads any file on the system (subject to worker isolation and mcp-sudo.yaml permissions).",
				"mimeType":    "text/plain",
			},
			map[string]interface{}{
				"uriTemplate": "devices://{type}",
				"name":        "Hardware Devices",
				"description": "Hardware device metadata. Valid types: usb, pci, dmi",
				"mimeType":    "application/json",
			},
		},
	}
}

func handleResourcesRead(session *Session, req JSONRPCRequest, resp *JSONRPCResponse) {
	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(req.Params, &params); err == nil {
		var content string
		var readErr error
		mimeType := "text/plain"

		// Pre-authorization check for strictly privileged resources
		if strings.HasPrefix(params.URI, "devices://") || strings.HasPrefix(params.URI, "kernel://") {
			if !sudoConfig.CanReadResourceAsRoot(session.User, params.URI, "*") {
				resp.Error = map[string]interface{}{"code": -32603, "message": fmt.Sprintf("resource %s is strictly accessible only in privileged mode", params.URI)}
				data, _ := json.Marshal(resp)
				session.Event <- string(data)
				return
			}
		}

		// 1. Check cache first!
		if cachedContent, cachedMimeType, hit := resourceCache.Get(params.URI); hit {
			resp.Result = map[string]interface{}{
				"contents": []interface{}{
					map[string]interface{}{
						"uri":      params.URI,
						"mimeType": cachedMimeType,
						"text":     cachedContent,
					},
				},
			}
			data, _ := json.Marshal(resp)
			session.Event <- string(data)
			return
		}

		if params.URI == "os://uname" {
			var uts syscall.Utsname
			if err := syscall.Uname(&uts); err != nil {
				readErr = fmt.Errorf("syscall.Uname failed: %v", err)
			} else {
				content = fmt.Sprintf("Sysname: %s\nNodename: %s\nRelease: %s\nVersion: %s\nMachine: %s",
					charsToString(uts.Sysname[:]),
					charsToString(uts.Nodename[:]),
					charsToString(uts.Release[:]),
					charsToString(uts.Version[:]),
					charsToString(uts.Machine[:]),
				)
			}
		} else if params.URI == "os://release" {
			data, err := os.ReadFile("/etc/os-release")
			if err != nil {
				readErr = fmt.Errorf("failed to read /etc/os-release: %v", err)
			} else {
				content = string(data)
			}
		} else if params.URI == "os://hostname" {
			hostname, err := os.Hostname()
			if err != nil {
				readErr = fmt.Errorf("os.Hostname failed: %v", err)
			} else {
				content = hostname
			}
		} else if params.URI == "network://interfaces" {
			ifaces, err := net.Interfaces()
			if err != nil {
				readErr = fmt.Errorf("net.Interfaces failed: %v", err)
			} else {
				var resultList []map[string]interface{}
				for _, iface := range ifaces {
					addrs, _ := iface.Addrs()
					var addrList []string
					for _, addr := range addrs {
						addrList = append(addrList, addr.String())
					}
					resultList = append(resultList, map[string]interface{}{
						"index":     iface.Index,
						"name":      iface.Name,
						"mac":       iface.HardwareAddr.String(),
						"mtu":       iface.MTU,
						"flags":     iface.Flags.String(),
						"addresses": addrList,
					})
				}
				b, _ := json.MarshalIndent(resultList, "", "  ")
				content = string(b)
				mimeType = "application/json"
			}
		} else if strings.HasPrefix(params.URI, "file://") {
			// Handle dynamic URN via Isolated Worker!
			rawPath := strings.TrimPrefix(params.URI, "file://")

			var path string
			var toolName string

			if strings.HasSuffix(rawPath, "/stat") {
				path = strings.TrimSuffix(rawPath, "/stat")
				toolName = "files/stat"
			} else if strings.HasSuffix(rawPath, "/content") {
				path = strings.TrimSuffix(rawPath, "/content")
				toolName = "files/content"
			} else if strings.HasSuffix(rawPath, "/type") {
				path = strings.TrimSuffix(rawPath, "/type")
				toolName = "files/filetype"
			} else {
				readErr = fmt.Errorf("invalid file URI: must end in /stat, /content, or /type")
				goto SendResponse
			}

			// Check mcp-sudo.yaml to see if this user is allowed to read THIS file as root
			isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, "file://", path)

			// We map it to the corresponding internal tool for the spawner
			argsJSON, _ := json.Marshal(map[string]interface{}{
				"path": path,
			})

			content, readErr = worker.SpawnWorker(session.User, toolName, argsJSON, isPrivileged, sudoConfig, 30)
		} else if params.URI == "devices://usb" {
			isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, params.URI, "*")
			if !isPrivileged {
				readErr = fmt.Errorf("resource %s is strictly accessible only in privileged mode", params.URI)
			} else {
				content, readErr = worker.SpawnWorker(session.User, "read_usb", []byte("{}"), isPrivileged, sudoConfig, 30)
				mimeType = "application/json"
			}
		} else if params.URI == "devices://pci" {
			isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, params.URI, "*")
			if !isPrivileged {
				readErr = fmt.Errorf("resource %s is strictly accessible only in privileged mode", params.URI)
			} else {
				content, readErr = worker.SpawnWorker(session.User, "read_pci", []byte("{}"), isPrivileged, sudoConfig, 30)
				mimeType = "application/json"
			}
		} else if params.URI == "devices://dmi" {
			isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, params.URI, "*")
			if !isPrivileged {
				readErr = fmt.Errorf("resource %s is strictly accessible only in privileged mode", params.URI)
			} else {
				content, readErr = worker.SpawnWorker(session.User, "read_dmi", []byte("{}"), isPrivileged, sudoConfig, 30)
				mimeType = "application/json"
			}
		} else if params.URI == "kernel://modules" {
			isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, params.URI, "*")
			if !isPrivileged {
				readErr = fmt.Errorf("resource %s is strictly accessible only in privileged mode", params.URI)
			} else {
				content, readErr = worker.SpawnWorker(session.User, "read_modules", []byte("{}"), isPrivileged, sudoConfig, 30)
				mimeType = "application/json"
			}
		} else if params.URI == "network://routes" {
			isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, params.URI, "*")
			content, readErr = worker.SpawnWorker(session.User, "read_routes", []byte("{}"), isPrivileged, sudoConfig, 30)
			mimeType = "application/json"
		} else {
			readErr = fmt.Errorf("unknown resource: %s", params.URI)
		}

	SendResponse:
		if readErr != nil {
			resp.Error = map[string]interface{}{"code": -32603, "message": readErr.Error()}
		} else {
			// Cache the result if applicable
			if params.URI == "devices://dmi" || params.URI == "os://hostname" {
				resourceCache.Set(params.URI, content, mimeType, 24*time.Hour)
			} else if params.URI == "devices://pci" {
				resourceCache.Set(params.URI, content, mimeType, 1*time.Hour)
			} else if params.URI == "os://release" || params.URI == "os://uname" {
				resourceCache.Set(params.URI, content, mimeType, 1*time.Hour)
			} else if params.URI == "devices://usb" || params.URI == "kernel://modules" {
				resourceCache.Set(params.URI, content, mimeType, 60*time.Second)
			} else if params.URI == "network://interfaces" || params.URI == "network://routes" {
				resourceCache.Set(params.URI, content, mimeType, 5*time.Second)
			}

			resp.Result = map[string]interface{}{
				"contents": []interface{}{
					map[string]interface{}{
						"uri":      params.URI,
						"mimeType": mimeType,
						"text":     content,
					},
				},
			}
		}
	} else {
		resp.Error = map[string]interface{}{"code": -32602, "message": "Invalid params"}
	}
}

func handleToolsList(session *Session, resp *JSONRPCResponse) {
	// Dynamically generate the tools list based on sudo rules.
	listDesc := "Lists contents of a directory."
	if sudoConfig.CanRunAsRoot(session.User, "files/list") {
		listDesc += " (Hint: You are authorized to run this tool as root. Use 'privileged: true' if you receive permission denied errors on sensitive paths)."
	}

	dfDesc := "Returns disk space statistics (df -h)."
	if sudoConfig.CanRunAsRoot(session.User, "disks/free") {
		dfDesc += " (Authorized for 'privileged: true')"
	}

	duDesc := "Calculates the total disk space utilized by a specific directory (du -sh)."
	if sudoConfig.CanRunAsRoot(session.User, "disks/usage") {
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
				"description": "Lists running processes on the system.",
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
				"description": "Lists active network connections and listening ports.",
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
				"description": "Returns memory and swap utilization information.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"detailed":      map[string]interface{}{"type": "boolean", "description": "Set to true to return raw /proc/meminfo instead of summary"},
					},
				},
			},
			map[string]interface{}{
				"name":        "cpu/list",
				"tools_group": "cpu",
				"description": "Retrieves CPU topology and architecture.",
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
				"description": "Retrieves system load averages (1m, 5m, 15m).",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"}},
				},
			},
			map[string]interface{}{
				"name":        "disks/list",
				"tools_group": "disks",
				"description": "Lists block devices.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
						"all":           map[string]interface{}{"type": "boolean", "description": "Include empty devices"},
					},
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

func handleToolsCall(session *Session, req JSONRPCRequest, resp *JSONRPCResponse) {
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
			"disks/free":          true,
			"disks/usage":         true,
			"processes/list":      true,
			"processes/delete":    true,
			"network/connections": true,
			"network/nslookup":    true,
			"network/curl":        true,
			"network/arp":         true,
			"network/ping":        true,
			"memory/usage":        true,
			"cpu/list":            true,
			"cpu/load-average":    true,
			"disks/list": true,
			"system/os-release":   true,
		}

		if params.Name == "auth/sudo-rules" {
			// No need to spawn an isolated worker to read our own memory config
			resultText, execErr = sudorules.SudoRules(params.Arguments, session.User, sudoConfig)

		} else if standardWorkers[params.Name] {

			// Standard privileged check payload
			var baseArgs struct {
				Privileged bool   `json:"privileged"`
				Path       string `json:"path"`
			}
			_ = json.Unmarshal(params.Arguments, &baseArgs)

			if (params.Name == "files/list" || params.Name == "files/read" || params.Name == "files/create" || params.Name == "files/update" || params.Name == "files/find") && baseArgs.Privileged {
				allowedPaths := sudoConfig.GetAllowedPaths(session.User, params.Name)
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
			executionTimeout := daemonConfig.Worker.TimeoutSeconds
			if toolCfg, ok := daemonConfig.Tools[params.Name]; ok && toolCfg.TimeoutSeconds > 0 {
				executionTimeout = toolCfg.TimeoutSeconds
			}

			if execErr == nil {
				// Use the Ephemeral Worker Spawner with caching for heavy tools
				if params.Name == "disks/usage" {
					cacheKey := fmt.Sprintf("%s:%s:%t", session.User, string(params.Arguments), baseArgs.Privileged)

					cacheMu.RLock()
					entry, ok := rpcCache[cacheKey]
					cacheMu.RUnlock()

					if ok && time.Now().Before(entry.expiresAt) {
						resultText = entry.result
					} else {
						v, err, _ := requestGroup.Do(cacheKey, func() (interface{}, error) {
							res, exErr := worker.SpawnWorker(session.User, params.Name, params.Arguments, baseArgs.Privileged, sudoConfig, executionTimeout)
							if exErr == nil {
								cacheMu.Lock()
								rpcCache[cacheKey] = cacheEntry{
									result:    res,
									expiresAt: time.Now().Add(60 * time.Second),
								}
								cacheMu.Unlock()
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
					resultText, execErr = worker.SpawnWorker(session.User, params.Name, params.Arguments, baseArgs.Privileged, sudoConfig, executionTimeout)
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
