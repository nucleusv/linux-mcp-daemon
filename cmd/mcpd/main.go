package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/auth"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/cache"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/devices/dmi"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/devices/pci"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/devices/usb"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/kernel/modules"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/network/routes"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/auth/get/sudo_rules"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/cpu/get/info"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/cpu/get/load_average"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/get/block-devices"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/get/free"
	disk_usage "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/get/usage"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/files/get/list_of_files"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/files/read/file"
	mem_usage "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/memory/get/usage"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/get/arp"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/get/connections"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/get/curl"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/get/nslookup"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/get/ping"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/processes/delete/process"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/processes/get/processes"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/system/get/os_release"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/worker"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int `yaml:"port"`
		TLS  struct {
			Enabled  bool   `yaml:"enabled"`
			Port     int    `yaml:"port"`
			CertFile string `yaml:"cert_file"`
			KeyFile  string `yaml:"key_file"`
		} `yaml:"tls"`
	} `yaml:"server"`
	RateLimits struct {
		DefaultRPS   float64 `yaml:"default_rps"`
		DefaultBurst int     `yaml:"default_burst"`
	} `yaml:"rate_limits"`
	Worker struct {
		TimeoutSeconds int `yaml:"timeout_seconds"`
	} `yaml:"worker"`
	Tools map[string]struct {
		TimeoutSeconds int `yaml:"timeout_seconds"`
	} `yaml:"tools"`
	Users []struct {
		Username string `yaml:"username"`
		Token    string `yaml:"token"`
	} `yaml:"users"`
}

var (
	daemonConfig   Config
	sudoConfig     *config.SudoConfig
	limiterManager *auth.LimiterManager
	
	sessions       = make(map[string]*Session)
	sessionsMu     sync.RWMutex
	sessionCounter int64

	requestGroup  singleflight.Group
	rpcCache      = make(map[string]cacheEntry)
	cacheMu       sync.RWMutex
	resourceCache = cache.NewTTLCache()
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

type cacheEntry struct {
	result    string
	expiresAt time.Time
}

type Session struct {
	ID    string
	User  string
	Event chan string
}

// MCP JSON-RPC Structures
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError,omitempty"`
}

func loadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &daemonConfig)
}

func main() {
	// ==========================================
	// WORKER MODE (Ephemeral Execution)
	// ==========================================
	if len(os.Args) > 1 && os.Args[1] == "worker" {
		if len(os.Args) < 4 {
			log.Fatalf("Usage: mcpd worker <tool_name> <json_args>")
		}
		toolName := os.Args[2]
		toolArgs := []byte(os.Args[3])

		var result string
		var err error

		if toolName == "list_files" {
			result, err = list_of_files.ListOfFiles(toolArgs)
		} else if toolName == "get_free" {
			result, err = free.Free(toolArgs)
		} else if toolName == "get_usage" {
			result, err = disk_usage.Usage(toolArgs)
		} else if toolName == "list_processes" {
			result, err = processes.Processes(toolArgs)
		} else if toolName == "delete_process" {
			result, err = process.DeleteProcess(toolArgs)
		} else if toolName == "list_connections" {
			result, err = connections.Connections(toolArgs)
		} else if toolName == "nslookup" {
			result, err = nslookup.Nslookup(toolArgs)
		} else if toolName == "curl" {
			result, err = curl.Curl(toolArgs)
		} else if toolName == "arp" {
			result, err = arp.ARP(toolArgs)
		} else if toolName == "ping" {
			result, err = ping.Ping(toolArgs)
		} else if toolName == "memory_usage" {
			result, err = mem_usage.Usage(toolArgs)
		} else if toolName == "get_info" {
			result, err = info.Info(toolArgs)
		} else if toolName == "load_average" {
			result, err = load_average.LoadAverage(toolArgs)
		} else if toolName == "block_devices" {
			result, err = blockdevices.GetBlockDevices(toolArgs)
		} else if toolName == "get_os_release" {
			result, err = os_release.OSRelease(toolArgs)
		} else if toolName == "read_file" {
			result, err = file.ReadFile(toolArgs)
		} else if toolName == "read_usb" {
			result, err = usb.ReadUSB(toolArgs)
		} else if toolName == "read_pci" {
			result, err = pci.ReadPCI(toolArgs)
		} else if toolName == "read_dmi" {
			result, err = dmi.ReadDMI(toolArgs)
		} else if toolName == "read_modules" {
			result, err = modules.ReadModules(toolArgs)
		} else if toolName == "read_routes" {
			result, err = routes.ReadRoutes(toolArgs)
		} else {
			log.Fatalf("Unknown tool: %s", toolName)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Print(result)
		os.Exit(0)
	}

	// ==========================================
	// MASTER DAEMON MODE
	// ==========================================
	if err := loadConfig("configs/daemon.yaml"); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var err error
	sudoConfig, err = config.LoadSudoConfig("configs/mcp-sudo.yaml")
	if err != nil {
		log.Fatalf("Failed to load mcp-sudo.yaml: %v", err)
	}

	limiterManager = auth.NewLimiterManager(daemonConfig.RateLimits.DefaultRPS, daemonConfig.RateLimits.DefaultBurst)

	addr := fmt.Sprintf(":%d", daemonConfig.Server.Port)
	if daemonConfig.Server.Port == 0 {
		addr = ":9090"
	}
	if daemonConfig.Worker.TimeoutSeconds == 0 {
		daemonConfig.Worker.TimeoutSeconds = 30
	}

	http.HandleFunc("/sse", handleSSE)
	http.HandleFunc("/message", handleMessage)

	// Serve documentation
	fs := http.FileServer(http.Dir("docs/website/build"))
	http.Handle("/docs/", http.StripPrefix("/docs/", fs))

	if daemonConfig.Server.TLS.Enabled {
		tlsAddr := fmt.Sprintf(":%d", daemonConfig.Server.TLS.Port)
		if daemonConfig.Server.TLS.Port == 0 {
			tlsAddr = ":9443"
		}

		go func() {
			log.Printf("Starting Linux MCP Daemon (HTTPS SSE Transport) on %s\n", tlsAddr)
			if err := http.ListenAndServeTLS(tlsAddr, daemonConfig.Server.TLS.CertFile, daemonConfig.Server.TLS.KeyFile, nil); err != nil {
				log.Fatalf("Daemon TLS crashed: %v", err)
			}
		}()
	}

	log.Printf("Starting Linux MCP Daemon (HTTP SSE Transport) on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Daemon crashed: %v", err)
	}
}

func authenticateRequest(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", false
	}
	providedToken := strings.TrimPrefix(authHeader, "Bearer ")
	for _, user := range daemonConfig.Users {
		if user.Token == providedToken {
			return user.Username, true
		}
	}
	return "", false
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	username, ok := authenticateRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	sessionUser := username

	sessionsMu.Lock()
	sessionCounter++
	uniqueSessionID := fmt.Sprintf("%s-%d", token, sessionCounter)
	
	session := &Session{
		ID:    uniqueSessionID,
		User:  sessionUser,
		Event: make(chan string, 10),
	}
	sessions[uniqueSessionID] = session
	sessionsMu.Unlock()

	log.Printf("SSE connection established for user: %s (Session: %s)", sessionUser, uniqueSessionID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	fmt.Fprintf(w, "event: endpoint\ndata: /message?session_id=%s\n\n", uniqueSessionID)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	defer func() {
		sessionsMu.Lock()
		delete(sessions, uniqueSessionID)
		sessionsMu.Unlock()
		close(session.Event)
		log.Printf("SSE connection closed for user: %s (Session: %s)", sessionUser, uniqueSessionID)
	}()

	for {
		select {
		case msg := <-session.Event:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-r.Context().Done():
			log.Printf("SSE connection closed for user: %s", username)
			return
		}
	}
}

func handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := authenticateRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "Missing session_id in query parameters", http.StatusBadRequest)
		return
	}

	sessionsMu.RLock()
	session, exists := sessions[sessionID]
	sessionsMu.RUnlock()

	if !exists {
		http.Error(w, "No active SSE session", http.StatusBadRequest)
		return
	}

	if !limiterManager.Allow(username) {
		log.Printf("[THROTTLED] User %s exceeded rate limits", username)
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("Failed to unmarshal JSON-RPC: %v", err)
		http.Error(w, "Invalid JSON-RPC", http.StatusBadRequest)
		return
	}

	log.Printf("Received JSON-RPC method: %s for session %s", req.Method, sessionID)

	go processJSONRPC(session, req)

	w.WriteHeader(http.StatusAccepted)
}

func processJSONRPC(session *Session, req JSONRPCRequest) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	if req.Method == "initialize" {
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
	} else if req.Method == "notifications/initialized" {
		// This is a client notification, not a request. The server should not send a response.
		return
	} else if req.Method == "ping" {
		resp.Result = map[string]interface{}{}
	} else if req.Method == "resources/list" {
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
	} else if req.Method == "resources/templates/list" {
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
	} else if req.Method == "resources/read" {
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
				path := strings.TrimPrefix(params.URI, "file://")

				// Check mcp-sudo.yaml to see if this user is allowed to read THIS file as root
				isPrivileged := sudoConfig.CanReadResourceAsRoot(session.User, "file://", path)

				// We map it to a "read_file" internal tool for the spawner
				argsJSON, _ := json.Marshal(map[string]interface{}{
					"path": path,
				})

				content, readErr = worker.SpawnWorker(session.User, "read_file", argsJSON, isPrivileged, sudoConfig, 30)
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
				readErr = fmt.Errorf("unsupported resource URI scheme: %s", params.URI)
			}

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
	} else if req.Method == "tools/list" {
		// Dynamically generate the tools list based on sudo rules.
		listDesc := "Lists contents of a directory."
		if sudoConfig.CanRunAsRoot(session.User, "get_list_of_files") {
			listDesc += " (Hint: You are authorized to run this tool as root. Use 'privileged: true' if you receive permission denied errors on sensitive paths)."
		}
		
		dfDesc := "Returns disk space statistics (df -h)."
		if sudoConfig.CanRunAsRoot(session.User, "get_free") {
			dfDesc += " (Authorized for 'privileged: true')"
		}
		
		duDesc := "Calculates the total disk space utilized by a specific directory (du -sh)."
		if sudoConfig.CanRunAsRoot(session.User, "get_usage") {
			duDesc += " (Authorized for 'privileged: true' to traverse protected subdirectories)"
		}

		toolsList := map[string]interface{}{
			"tools": []interface{}{
				map[string]interface{}{
					"name": "list_files",
					"tools_group": "files",
					"description": listDesc,
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
							"path":       map[string]interface{}{"type": "string", "description": "Absolute path to list"},
							"privileged": map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
						},
						"required": []string{"path"},
					},
				},
				map[string]interface{}{
					"name": "get_free",
					"tools_group": "disks",
					"description": dfDesc,
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
							"path":           map[string]interface{}{"type": "string", "description": "Absolute path to check"},
							"inodes":         map[string]interface{}{"type": "boolean", "description": "List inode information instead of block usage (-i)"},
							"human_readable": map[string]interface{}{"type": "boolean", "description": "Print sizes in powers of 1024 (-h)"},
							"privileged":     map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
						},
						"required": []string{"path"},
					},
				},
				map[string]interface{}{
					"name": "get_usage",
					"tools_group": "disks",
					"description": duDesc,
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
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
					"name": "list_processes",
					"tools_group": "processes",
					"description": "Lists running processes on the system.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
							"user":       map[string]interface{}{"type": "string", "description": "Filter by username"},
							"sort_by":    map[string]interface{}{"type": "string", "description": "Sort by cpu, mem, or pid"},
							"limit":      map[string]interface{}{"type": "integer", "description": "Limit returned processes"},
							"privileged": map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
						},
					},
				},
				map[string]interface{}{
					"name": "delete_process",
					"tools_group": "processes",
					"description": "Terminates a specific process by PID.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
							"pid":        map[string]interface{}{"type": "integer", "description": "The PID to kill"},
							"signal":     map[string]interface{}{"type": "string", "description": "Signal to send (e.g., SIGTERM, SIGKILL)"},
							"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root to kill other user's processes"},
						},
						"required": []string{"pid"},
					},
				},
				map[string]interface{}{
					"name":        "nslookup",
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
					"name":        "curl",
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
					"name":        "arp",
					"description": "View the system ARP cache (IP to MAC address mappings).",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"interface": map[string]interface{}{"type": "string"},
						},
					},
				},
				map[string]interface{}{
					"name":        "ping",
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
					"name": "list_connections",
					"tools_group": "network",
					"description": "Lists active network connections and listening ports.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
							"state":      map[string]interface{}{"type": "string", "description": "Filter by TCP state (e.g., LISTEN, ESTABLISHED)"},
							"port":       map[string]interface{}{"type": "integer", "description": "Filter by port"},
							"privileged": map[string]interface{}{"type": "boolean", "description": "Run as root to see PIDs of other users"},
						},
					},
				},
				map[string]interface{}{
					"name": "memory_usage",
					"tools_group": "memory",
					"description": "Returns memory and swap utilization information.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
							"detailed": map[string]interface{}{"type": "boolean", "description": "Set to true to return raw /proc/meminfo instead of summary"},
						},
					},
				},
				map[string]interface{}{
					"name": "get_info",
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
					"name": "load_average",
					"tools_group": "cpu",
					"description": "Retrieves system load averages (1m, 5m, 15m).",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},},
					},
				},
				map[string]interface{}{
					"name": "block_devices",
					"tools_group": "disks",
					"description": "Lists block devices.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
							"all": map[string]interface{}{"type": "boolean", "description": "Include empty devices"},
						},
					},
				},
				map[string]interface{}{
					"name": "get_os_release",
					"tools_group": "system",
					"description": "Retrieves Linux distribution and kernel version.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},},
					},
				},
				map[string]interface{}{
					"name": "get_sudo_rules",
					"tools_group": "auth",
					"description": "Returns your authorized tools and privileges from mcp-sudo.yaml.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},},
					},
				},
				map[string]interface{}{
					"name": "nslookup",
					"tools_group": "network",
					"description": "Resolves a hostname to an IP address.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"host": map[string]interface{}{"type": "string", "description": "Hostname to resolve"},
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format"},
						},
						"required": []string{"host"},
					},
				},
				map[string]interface{}{
					"name": "curl",
					"tools_group": "network",
					"description": "Executes an HTTP GET request.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"url": map[string]interface{}{"type": "string", "description": "URL to fetch"},
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format"},
						},
						"required": []string{"url"},
					},
				},
				map[string]interface{}{
					"name": "arp",
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
					"name": "ping",
					"tools_group": "network",
					"description": "Sends ICMP echo requests.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"host": map[string]interface{}{"type": "string", "description": "Host to ping"},
							"count": map[string]interface{}{"type": "integer", "description": "Number of packets"},
							"output_format": map[string]interface{}{"type": "string", "description": "Desired output format"},
						},
						"required": []string{"host"},
					},
				},
			},
		}
		resp.Result = toolsList

	} else if req.Method == "tools/call" {
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			
			toolRes := ToolResult{}
			var resultText string
			var execErr error

			if params.Name == "get_sudo_rules" {
				// No need to spawn an isolated worker to read our own memory config
				resultText, execErr = sudo_rules.SudoRules(params.Arguments, session.User, sudoConfig)

			} else if params.Name == "list_files" || params.Name == "get_free" || params.Name == "get_usage" || params.Name == "list_processes" || params.Name == "delete_process" || params.Name == "get_interfaces" || params.Name == "list_connections" || params.Name == "nslookup" || params.Name == "curl" || params.Name == "arp" || params.Name == "ping" || params.Name == "memory_usage" || params.Name == "get_info" || params.Name == "load_average" || params.Name == "block_devices" || params.Name == "get_os_release" {
				
				// Standard privileged check payload
				var baseArgs struct {
					Privileged bool   `json:"privileged"`
					Path       string `json:"path"`
				}
				_ = json.Unmarshal(params.Arguments, &baseArgs)

				if params.Name == "get_list_of_files" && baseArgs.Privileged {
					allowedPaths := sudoConfig.GetAllowedPaths(session.User, "get_list_of_files")
					allowed := false
					for _, p := range allowedPaths {
						if strings.HasPrefix(baseArgs.Path, p) {
							allowed = true
							break
						}
					}
					if !allowed {
						execErr = fmt.Errorf("permission denied: path '%s' is not in your allowed paths for list_directory", baseArgs.Path)
					}
				}
				
				// Resolve execution timeout (check tool override, fallback to global worker default)
				executionTimeout := daemonConfig.Worker.TimeoutSeconds
				if toolCfg, ok := daemonConfig.Tools[params.Name]; ok && toolCfg.TimeoutSeconds > 0 {
					executionTimeout = toolCfg.TimeoutSeconds
				}

				if execErr == nil {
					// Use the Ephemeral Worker Spawner with caching for heavy tools
					if params.Name == "get_usage" {
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
					toolRes.Content = append(toolRes.Content, struct{Type string `json:"type"`; Text string `json:"text"`}{Type: "text", Text: execErr.Error()})
				} else {
					toolRes.Content = append(toolRes.Content, struct{Type string `json:"type"`; Text string `json:"text"`}{Type: "text", Text: resultText})
				}
				resp.Result = toolRes
			}

		} else {
			resp.Error = map[string]interface{}{"code": -32602, "message": "Invalid params"}
		}
	} else {
		resp.Error = map[string]interface{}{"code": -32601, "message": "Method not found"}
	}

	respBytes, _ := json.Marshal(resp)
	log.Printf("Sending response for %s: %s", req.Method, string(respBytes))
	session.Event <- string(respBytes)
}