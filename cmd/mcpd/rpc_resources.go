package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/network/interfaces"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/os/hostname"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/os/release"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/worker"
)

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
			map[string]interface{}{
				"uriTemplate": "network://interfaces/{name}",
				"name":        "Network Interface Detail",
				"description": "Detailed properties of a specific network interface.",
				"mimeType":    "application/json",
			},
			map[string]interface{}{
				"uriTemplate": "service://{name}/status",
				"name":        "Service Status",
				"description": "Exposes DBus service properties (ActiveState, LoadState, SubState).",
				"mimeType":    "application/json",
			},
			map[string]interface{}{
				"uriTemplate": "process://{pid}/{target}",
				"name":        "Process Introspection",
				"description": "Reads process metadata from procfs. Valid targets: status, cmdline, environ",
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
		case params.URI == "os://hostname":
			content, mimeType, readErr = hostname.Read()
		case strings.HasPrefix(params.URI, "network://interfaces"):
			targetName := strings.TrimPrefix(params.URI, "network://interfaces")
			targetName = strings.TrimPrefix(targetName, "/")
			content, mimeType, readErr = interfaces.Read(targetName)
		case strings.HasPrefix(params.URI, "file://"):
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
		case strings.HasPrefix(params.URI, "service://") && strings.HasSuffix(params.URI, "/status"):
			// service://kubelet.service/status
			serviceName := strings.TrimPrefix(params.URI, "service://")
			serviceName = strings.TrimSuffix(serviceName, "/status")

			isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, "service://", serviceName)
			
			argsJSON, _ := json.Marshal(map[string]interface{}{
				"service": serviceName,
			})
			content, readErr = worker.SpawnWorker(session.User, "services/status", argsJSON, isPrivileged, sudoConfig, 30)
			mimeType = "application/json"
		case strings.HasPrefix(params.URI, "process://"):
			// process://{pid}/status, process://{pid}/cmdline, process://{pid}/environ
			rawPath := strings.TrimPrefix(params.URI, "process://")
			parts := strings.SplitN(rawPath, "/", 2)
			if len(parts) != 2 {
				readErr = fmt.Errorf("invalid process URI format, expected process://{pid}/{target}")
				goto SendResponse
			}
			
			pidStr, target := parts[0], parts[1]
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				readErr = fmt.Errorf("invalid PID: %v", err)
				goto SendResponse
			}

			isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, "process://", pidStr)
			
			argsJSON, _ := json.Marshal(map[string]interface{}{
				"pid":    pid,
				"target": target,
			})
			content, readErr = worker.SpawnWorker(session.User, "processes/read", argsJSON, isPrivileged, sudoConfig, 30)
			if target != "status" {
				mimeType = "application/json"
			}
		case params.URI == "devices://usb":
			isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, params.URI, "*")
			if !isPrivileged {
				readErr = fmt.Errorf("resource %s is strictly accessible only in privileged mode", params.URI)
			} else {
				content, readErr = worker.SpawnWorker(session.User, "read_usb", []byte("{}"), isPrivileged, sudoConfig, 30)
				mimeType = "application/json"
			}
		case params.URI == "devices://pci":
			isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, params.URI, "*")
			if !isPrivileged {
				readErr = fmt.Errorf("resource %s is strictly accessible only in privileged mode", params.URI)
			} else {
				content, readErr = worker.SpawnWorker(session.User, "read_pci", []byte("{}"), isPrivileged, sudoConfig, 30)
				mimeType = "application/json"
			}
		case params.URI == "devices://dmi":
			isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, params.URI, "*")
			if !isPrivileged {
				readErr = fmt.Errorf("resource %s is strictly accessible only in privileged mode", params.URI)
			} else {
				content, readErr = worker.SpawnWorker(session.User, "read_dmi", []byte("{}"), isPrivileged, sudoConfig, 30)
				mimeType = "application/json"
			}
		case params.URI == "kernel://modules":
			isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, params.URI, "*")
			if !isPrivileged {
				readErr = fmt.Errorf("resource %s is strictly accessible only in privileged mode", params.URI)
			} else {
				content, readErr = worker.SpawnWorker(session.User, "read_modules", []byte("{}"), isPrivileged, sudoConfig, 30)
				mimeType = "application/json"
			}
		case params.URI == "network://routes":
			isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, params.URI, "*")
			content, readErr = worker.SpawnWorker(session.User, "read_routes", []byte("{}"), isPrivileged, sudoConfig, 30)
			mimeType = "application/json"
		default:
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
