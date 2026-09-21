package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/auth"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/get_disk_free"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/get_disk_usage"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/get_sudo_rules"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/list_directory"
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
	
	sessions   = make(map[string]*Session)
	sessionsMu sync.RWMutex

	requestGroup singleflight.Group
	cache        = make(map[string]cacheEntry)
	cacheMu      sync.RWMutex
)

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

		if toolName == "list_directory" {
			result, err = list_directory.ListDirectory(toolArgs)
		} else if toolName == "get_disk_free" {
			result, err = get_disk_free.GetDiskFree(toolArgs)
		} else if toolName == "get_disk_usage" {
			result, err = get_disk_usage.GetDiskUsage(toolArgs)
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

	sessionID := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	session := &Session{
		ID:    sessionID,
		User:  username,
		Event: make(chan string, 10),
	}

	sessionsMu.Lock()
	sessions[sessionID] = session
	sessionsMu.Unlock()

	defer func() {
		sessionsMu.Lock()
		delete(sessions, sessionID)
		sessionsMu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	fmt.Fprintf(w, "event: endpoint\ndata: /message\n\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	log.Printf("SSE connection established for user: %s", username)

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

	sessionID := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

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
		http.Error(w, "Invalid JSON-RPC", http.StatusBadRequest)
		return
	}

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
				"tools": map[string]interface{}{},
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
	} else if req.Method == "tools/list" {
		// Dynamically generate the tools list based on sudo rules.
		listDesc := "Lists contents of a directory."
		if sudoConfig.CanRunAsRoot(session.User, "list_directory") {
			listDesc += " (Hint: You are authorized to run this tool as root. Use 'privileged: true' if you receive permission denied errors on sensitive paths)."
		}
		
		dfDesc := "Returns disk space statistics (df -h)."
		if sudoConfig.CanRunAsRoot(session.User, "get_disk_free") {
			dfDesc += " (Authorized for 'privileged: true')"
		}
		
		duDesc := "Calculates the total disk space utilized by a specific directory (du -sh)."
		if sudoConfig.CanRunAsRoot(session.User, "get_disk_usage") {
			duDesc += " (Authorized for 'privileged: true' to traverse protected subdirectories)"
		}

		toolsList := map[string]interface{}{
			"tools": []interface{}{
				map[string]interface{}{
					"name": "list_directory",
					"tools_group": "files",
					"description": listDesc,
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path":       map[string]interface{}{"type": "string", "description": "Absolute path to list"},
							"privileged": map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
						},
						"required": []string{"path"},
					},
				},
				map[string]interface{}{
					"name": "get_disk_free",
					"tools_group": "disks",
					"description": dfDesc,
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path":           map[string]interface{}{"type": "string", "description": "Absolute path to check"},
							"inodes":         map[string]interface{}{"type": "boolean", "description": "List inode information instead of block usage (-i)"},
							"human_readable": map[string]interface{}{"type": "boolean", "description": "Print sizes in powers of 1024 (-h)"},
							"privileged":     map[string]interface{}{"type": "boolean", "description": "Set to true to run as root"},
						},
						"required": []string{"path"},
					},
				},
				map[string]interface{}{
					"name": "get_disk_usage",
					"tools_group": "disks",
					"description": duDesc,
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
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
					"name": "get_sudo_rules",
					"tools_group": "auth",
					"description": "Returns your authorized tools and privileges from mcp-sudo.yaml.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{},
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
				resultText, execErr = get_sudo_rules.GetSudoRules(session.User, sudoConfig)

			} else if params.Name == "list_directory" || params.Name == "get_disk_free" || params.Name == "get_disk_usage" {
				
				// All these tools share the 'privileged' boolean and 'path' string in their arguments
				var baseArgs struct {
					Privileged bool   `json:"privileged"`
					Path       string `json:"path"`
				}
				_ = json.Unmarshal(params.Arguments, &baseArgs)

				if params.Name == "list_directory" && baseArgs.Privileged {
					allowedPaths := sudoConfig.GetAllowedPaths(session.User, "list_directory")
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
					if params.Name == "get_disk_usage" {
						cacheKey := fmt.Sprintf("%s:%s:%t", session.User, string(params.Arguments), baseArgs.Privileged)
						
						cacheMu.RLock()
						entry, ok := cache[cacheKey]
						cacheMu.RUnlock()

						if ok && time.Now().Before(entry.expiresAt) {
							resultText = entry.result
						} else {
							v, err, _ := requestGroup.Do(cacheKey, func() (interface{}, error) {
								res, exErr := worker.SpawnWorker(session.User, params.Name, params.Arguments, baseArgs.Privileged, sudoConfig, executionTimeout)
								if exErr == nil {
									cacheMu.Lock()
									cache[cacheKey] = cacheEntry{
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
	session.Event <- string(respBytes)
}