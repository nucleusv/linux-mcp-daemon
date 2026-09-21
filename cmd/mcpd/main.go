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

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/auth"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	RateLimits struct {
		DefaultRPS   float64 `yaml:"default_rps"`
		DefaultBurst int     `yaml:"default_burst"`
	} `yaml:"rate_limits"`
	Users []struct {
		Username string `yaml:"username"`
		Token    string `yaml:"token"`
	} `yaml:"users"`
}

var (
	daemonConfig   Config
	limiterManager *auth.LimiterManager
	
	sessions   = make(map[string]*Session)
	sessionsMu sync.RWMutex
)

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
	if err := loadConfig("configs/daemon.yaml"); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Rate Limiter
	limiterManager = auth.NewLimiterManager(daemonConfig.RateLimits.DefaultRPS, daemonConfig.RateLimits.DefaultBurst)

	addr := fmt.Sprintf(":%d", daemonConfig.Server.Port)
	if daemonConfig.Server.Port == 0 {
		addr = ":9090" // fallback
	}

	log.Printf("Starting Linux MCP Daemon (SSE Transport) on %s\n", addr)

	http.HandleFunc("/sse", handleSSE)
	http.HandleFunc("/message", handleMessage)

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

	// We use the token as a simple session ID for this proof of concept.
	// In production, you'd generate a secure random UUID per connection.
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

	// Send the initial endpoint event required by MCP
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

	// 1. Check Rate Limits
	if !limiterManager.Allow(username) {
		log.Printf("[THROTTLED] User %s exceeded rate limits", username)
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// 2. Parse JSON-RPC
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

	// We process the request asynchronously and acknowledge the HTTP POST immediately.
	go processJSONRPC(session, req)

	w.WriteHeader(http.StatusAccepted)
}

func processJSONRPC(session *Session, req JSONRPCRequest) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	if req.Method == "tools/call" {
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			
			// Route to tools
			if params.Name == "list_directory" {
				resultText, err := tools.ListDirectory(params.Arguments)
				
				toolRes := ToolResult{}
				if err != nil {
					toolRes.IsError = true
					toolRes.Content = append(toolRes.Content, struct{Type string `json:"type"`; Text string `json:"text"`}{Type: "text", Text: err.Error()})
				} else {
					toolRes.Content = append(toolRes.Content, struct{Type string `json:"type"`; Text string `json:"text"`}{Type: "text", Text: resultText})
				}
				resp.Result = toolRes
			} else {
				resp.Error = map[string]interface{}{"code": -32601, "message": "Method not found"}
			}
		} else {
			resp.Error = map[string]interface{}{"code": -32602, "message": "Invalid params"}
		}
	} else {
		// For now, only support tools/call
		resp.Error = map[string]interface{}{"code": -32601, "message": "Method not found"}
	}

	// Send response back over SSE
	respBytes, _ := json.Marshal(resp)
	session.Event <- string(respBytes)
}