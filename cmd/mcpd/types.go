package main

import (
	"os"
	"sync"

	"golang.org/x/sync/singleflight"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/auth"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/cache"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/rpc"
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
		// Containerized indicates this daemon process itself runs inside a
		// container with its own private root filesystem (e.g. Kubernetes),
		// as opposed to running directly on the host with no container
		// boundary. When true, every privileged (root) worker call also
		// joins the real host's mount namespace before running - see
		// internal/worker/hostns.go. Set false when mcpd runs directly on
		// the host.
		Containerized bool `yaml:"containerized"`
	} `yaml:"worker"`
	Tools map[string]struct {
		TimeoutSeconds int `yaml:"timeout_seconds"`
	} `yaml:"tools"`
	Users []struct {
		Username string `yaml:"username"`
		// Token is the legacy plaintext field. New/rotated users
		// (via `linuxctl create|update mcpd user`) use TokenSalt+TokenHash
		// instead - see authenticateRequest in http.go. Both are supported
		// simultaneously so migration doesn't require a hard cutover.
		Token     string `yaml:"token,omitempty"`
		TokenSalt string `yaml:"token_salt,omitempty"`
		TokenHash string `yaml:"token_hash,omitempty"`
		CreatedAt string `yaml:"created_at,omitempty"`
		// PinnedUID and OSUID implement trust-on-first-use OS identity
		// pinning - see cmd/mcpd/uid_pin.go for the full mechanism, rationale,
		// and its known limitation (it does not reliably catch a username
		// being reused for a different real person if the OS happens to
		// reissue the exact same UID, which useradd's gap-filling behavior
		// makes plausible - see ARCHITECTURE.md). Never set these fields by
		// hand; they're written only by the daemon itself.
		PinnedUID string `yaml:"pinned_uid,omitempty"`
		OSUID     string `yaml:"os_uid,omitempty"`
	} `yaml:"users"`
}

// daemonConfigPath is set once in main() alongside loadConfig, and used by
// uid_pin.go to persist pinned_uid/os_uid updates back to the same file
// mcpd loaded them from.
var daemonConfigPath string

var (
	daemonConfig   Config
	sudoConfig     *config.SudoConfig
	limiterManager *auth.LimiterManager
	rpcHandler     *rpc.RPCHandler

	sessions       = make(map[string]*rpc.Session)
	sessionsMu     sync.RWMutex
	sessionCounter int64

	requestGroup  singleflight.Group
	rpcCache      = make(map[string]rpc.CacheEntry)
	cacheMu       sync.RWMutex
	resourceCache = cache.NewTTLCache()
)




func loadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &daemonConfig)
}
