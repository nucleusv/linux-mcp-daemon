package main

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/auth"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/cache"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
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
