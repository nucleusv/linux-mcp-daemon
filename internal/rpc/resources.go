package rpc

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/resources/os/release"
	"github.com/nucleusv/linux-mcp-daemon/internal/resources/system/hostname"
	"github.com/nucleusv/linux-mcp-daemon/internal/resources/system/locale"
	"github.com/nucleusv/linux-mcp-daemon/internal/resources/system/timezone"
	"github.com/nucleusv/linux-mcp-daemon/internal/resources/templates/disks"
	"github.com/nucleusv/linux-mcp-daemon/internal/resources/templates/file"
	"github.com/nucleusv/linux-mcp-daemon/internal/resources/templates/process"
	"github.com/nucleusv/linux-mcp-daemon/internal/resources/templates/service"
	"github.com/nucleusv/linux-mcp-daemon/internal/worker"
)

func charsToString(ca []int8) string {
	s := make([]byte, len(ca))
	var i int
	for ; i < len(ca); i++ {
		if ca[i] == 0 {
			break
		}
		s[i] = uint8(ca[i])
	}
	return string(s[:i])
}

func (h *RPCHandler) HandleResourcesList(resp *JSONRPCResponse) {
	// Enumerate static resources and wildcard templates
	resp.Result = map[string]interface{}{
		"resources": []interface{}{
			map[string]interface{}{
				"uri":           "os://uname",
				"name":          "OS Uname",
				"group":         "system",
				"linuxctl_verb": "uname",
				"description":   "Native system uname information (kernel version, node name). Hint: For CPU hardware architecture use cpu/list tool.",
				"mimeType":      "text/plain",
			},
			map[string]interface{}{
				"uri":           "os://release",
				"name":          "OS Release",
				"group":         "system",
				"linuxctl_verb": "release",
				"description":   "/etc/os-release information (distribution, version).",
				"mimeType":      "text/plain",
			},
			map[string]interface{}{
				"uri":           "system://hostname",
				"name":          "System Hostname",
				"group":         "system",
				"linuxctl_verb": "hostname",
				"description":   "Native system network hostname. Hint: To resolve IP addresses use network/nslookup tool.",
				"mimeType":      "text/plain",
			},
			map[string]interface{}{
				"uri":           "system://timezone",
				"name":          "System Timezone",
				"group":         "system",
				"linuxctl_verb": "timezone",
				"description":   "Configured IANA timezone (e.g. America/New_York) plus current local offset and time.",
				"mimeType":      "text/plain",
			},
			map[string]interface{}{
				"uri":           "system://locale",
				"name":          "System Locale",
				"group":         "system",
				"linuxctl_verb": "locale",
				"description":   "Configured locale settings (LANG, LC_*).",
				"mimeType":      "text/plain",
			},
			map[string]interface{}{
				"uri":           "network://interfaces",
				"name":          "Network Interfaces",
				"group":         "network",
				"linuxctl_verb": "interfaces",
				"description":   "Network interfaces, assigned IP addresses, and detailed RX/TX traffic statistics for all interfaces.",
				"mimeType":      "application/json",
			},
			map[string]interface{}{
				"uri":           "network://routes",
				"name":          "Network Routes",
				"group":         "network",
				"linuxctl_verb": "routes",
				"description":   "IPv4 Routing Table (/proc/net/route). Hint: Use network/ping to test reachability.",
				"mimeType":      "application/json",
			},
			map[string]interface{}{
				"uri":           "devices://usb",
				"name":          "USB Devices",
				"group":         "devices",
				"linuxctl_verb": "usb",
				"description":   "Connected USB devices (lsusb equivalent). Lists vendors, products, and bus mapping.",
				"mimeType":      "application/json",
			},
			map[string]interface{}{
				"uri":           "devices://pci",
				"name":          "PCI Devices",
				"group":         "devices",
				"linuxctl_verb": "pci",
				"description":   "Connected PCI devices (lspci equivalent). Includes network cards, GPUs, and controllers.",
				"mimeType":      "application/json",
			},
			map[string]interface{}{
				"uri":           "devices://dmi",
				"name":          "DMI Hardware Info",
				"group":         "devices",
				"linuxctl_verb": "dmi",
				"description":   "Desktop Management Interface info (lshw/hwinfo equivalent). Detailed hardware specifications (RAM banks, BIOS, chassis).",
				"mimeType":      "application/json",
			},
			map[string]interface{}{
				"uri":           "kernel://modules",
				"name":          "Kernel Modules",
				"group":         "kernel",
				"linuxctl_verb": "modules",
				"description":   "Loaded kernel drivers (lsmod equivalent). Hint: You can adjust kernel parameters via the kernel/system-control tool.",
				"mimeType":      "application/json",
			},
		},
	}
}

func (h *RPCHandler) HandleResourcesTemplatesList(resp *JSONRPCResponse) {
	resp.Result = map[string]interface{}{
		"resourceTemplates": []interface{}{
			map[string]interface{}{
				"uriTemplate": "file:///{path}",
				"name":        "File Reader",
				"group":       "files",
				"description": "Reads any file on the system. Append /stat for file metadata, /content for contents, or /type for file type.",
				"mimeType":    "text/plain",
			},
			map[string]interface{}{
				"uriTemplate": "devices://{type}",
				"name":        "Hardware Devices",
				"group":       "devices",
				"description": "Hardware device metadata. Valid types: usb, pci, dmi. Useful for inspecting attached physical hardware.",
				"mimeType":    "application/json",
			},
			map[string]interface{}{
				"uriTemplate":   "network://interfaces/{name}",
				"name":          "Network Interface Detail",
				"group":         "network",
				"linuxctl_verb": "interfaces",
				"description":   "Detailed properties and RX/TX traffic statistics of a specific network interface.",
				"mimeType":      "application/json",
			},
			map[string]interface{}{
				"uriTemplate":   "service://{name}/status",
				"name":          "Service Status",
				"group":         "system",
				"linuxctl_verb": "services",
				"description":   "Exposes DBus service properties (ActiveState, LoadState, SubState). Best used alongside the services/manage tool to check if a service actually started.",
				"mimeType":      "application/json",
			},
			map[string]interface{}{
				"uriTemplate": "disks://{name}/stats",
				"name":        "Disk I/O Statistics",
				"group":       "disks",
				"description": "Real-time I/O statistics for a specific block device (e.g. sda). Returns JSON.",
				"mimeType":    "application/json",
			},
			map[string]interface{}{
				"uriTemplate": "process://{pid}/{target}",
				"name":        "Process Introspection",
				"group":       "processes",
				"description": "Reads process metadata from procfs. Valid targets: status, cmdline, environ, limits (rlimits, e.g. max open files), open_files (fd table - files, sockets, pipes). Hint: Find PIDs using the processes/list tool first.",
				"mimeType":    "application/json",
			},
		},
	}
}

func (h *RPCHandler) HandleResourcesRead(session *Session, req JSONRPCRequest, resp *JSONRPCResponse) {
	// One snapshot per request: a concurrent reload can't change the rules
	// between the authorization check and the call it authorizes.
	sudoCfg := h.Sudo()
	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(req.Params, &params); err == nil {
		var content string
		var readErr error
		mimeType := "text/plain"

		// Pre-authorization check for strictly privileged resources
		if strings.HasPrefix(params.URI, "devices://") || strings.HasPrefix(params.URI, "kernel://") {
			if !sudoCfg.CanReadResourceAsRoot(session.User, params.URI, "*") {
				resp.Error = map[string]interface{}{"code": -32603, "message": fmt.Sprintf("resource %s is strictly accessible only in privileged mode", params.URI)}
				data, _ := json.Marshal(resp)
				session.Event <- string(data)
				return
			}
		}

		// 1. Check cache first!
		if cachedContent, cachedMimeType, hit := h.ResourceCache.Get(params.URI); hit {
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

		switch {
		case params.URI == "os://uname":
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
		case params.URI == "os://release":
			content, mimeType, readErr = release.Read()
		case params.URI == "system://hostname":
			content, mimeType, readErr = hostname.Read()
		case params.URI == "system://timezone":
			content, mimeType, readErr = timezone.Read()
		case params.URI == "system://locale":
			content, mimeType, readErr = locale.Read()
		case strings.HasPrefix(params.URI, "network://interfaces"):
			targetName := strings.TrimPrefix(params.URI, "network://interfaces")
			targetName = strings.TrimPrefix(targetName, "/")
			// Read in an unprivileged worker, never in this (root) master
			// process - the master must not touch /proc or /sys itself.
			argsJSON, _ := json.Marshal(map[string]string{"name": targetName})
			content, readErr = worker.SpawnWorker(session.User, "read_interfaces", argsJSON, false, sudoCfg, 30)
			mimeType = "application/json"
		case strings.HasPrefix(params.URI, "file://"):
			content, mimeType, readErr = file.Handle(params.URI, session.User, sudoCfg)
		case strings.HasPrefix(params.URI, "service://") && strings.HasSuffix(params.URI, "/status"):
			content, mimeType, readErr = service.Handle(params.URI, session.User, sudoCfg)
		case strings.HasPrefix(params.URI, "process://"):
			content, mimeType, readErr = process.Handle(params.URI, session.User, sudoCfg)
		case params.URI == "devices://usb":
			isPrivileged := sudoCfg.CanReadResourceAsRoot(session.User, params.URI, "*")
			if !isPrivileged {
				readErr = fmt.Errorf("resource %s is strictly accessible only in privileged mode", params.URI)
			} else {
				content, readErr = worker.SpawnWorker(session.User, "read_usb", []byte("{}"), isPrivileged, sudoCfg, 30)
				mimeType = "application/json"
			}
		case params.URI == "devices://pci":
			isPrivileged := sudoCfg.CanReadResourceAsRoot(session.User, params.URI, "*")
			if !isPrivileged {
				readErr = fmt.Errorf("resource %s is strictly accessible only in privileged mode", params.URI)
			} else {
				content, readErr = worker.SpawnWorker(session.User, "read_pci", []byte("{}"), isPrivileged, sudoCfg, 30)
				mimeType = "application/json"
			}
		case params.URI == "devices://dmi":
			isPrivileged := sudoCfg.CanReadResourceAsRoot(session.User, params.URI, "*")
			if !isPrivileged {
				readErr = fmt.Errorf("resource %s is strictly accessible only in privileged mode", params.URI)
			} else {
				content, readErr = worker.SpawnWorker(session.User, "read_dmi", []byte("{}"), isPrivileged, sudoCfg, 30)
				mimeType = "application/json"
			}
		case params.URI == "kernel://modules":
			isPrivileged := sudoCfg.CanReadResourceAsRoot(session.User, params.URI, "*")
			if !isPrivileged {
				readErr = fmt.Errorf("resource %s is strictly accessible only in privileged mode", params.URI)
			} else {
				content, readErr = worker.SpawnWorker(session.User, "read_modules", []byte("{}"), isPrivileged, sudoCfg, 30)
				mimeType = "application/json"
			}
		case params.URI == "network://routes":
			isPrivileged := sudoCfg.CanReadResourceAsRoot(session.User, params.URI, "*")
			content, readErr = worker.SpawnWorker(session.User, "read_routes", []byte("{}"), isPrivileged, sudoCfg, 30)
			mimeType = "application/json"
		case strings.HasPrefix(params.URI, "disks://") && strings.HasSuffix(params.URI, "/stats"):
			content, mimeType, readErr = disks.Handle(params.URI, session.User, sudoCfg)
		default:
			readErr = fmt.Errorf("unknown resource: %s", params.URI)
		}

		if readErr != nil {
			resp.Error = map[string]interface{}{"code": -32603, "message": readErr.Error()}
		} else {
			// Cache the result if applicable: 60 seconds for static
			// resources (long TTLs like the former 24h meant e.g. a changed
			// hostname went unseen for a day), 5 seconds for network state,
			// which changes often.
			switch params.URI {
			case "devices://dmi", "devices://pci", "devices://usb", "kernel://modules",
				"os://release", "os://uname", "system://hostname":
				h.ResourceCache.Set(params.URI, content, mimeType, 60*time.Second)
			case "network://interfaces", "network://routes":
				h.ResourceCache.Set(params.URI, content, mimeType, 5*time.Second)
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
