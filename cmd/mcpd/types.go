package main

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/singleflight"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/auth"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/cache"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/rpc"
)

// Config is daemon.yaml (see internal/config).
type Config = config.DaemonConfig

// configDir holds daemon.yaml, users.yaml and mcp-sudo.yaml, relative to
// mcpd's working directory.
const (
	configDir      = "configs"
	sudoConfigPath = configDir + "/" + config.SudoFile
)

var (
	// daemonConfig is replaced by reloadConfig (daemon/reload-config) and
	// its users updated in place by checkAndPinUID; both hold cfgMu, and
	// readers take cfgMu.RLock.
	daemonConfig Config
	// usersPath is the file daemonConfig.Users was loaded from (users.yaml,
	// or daemon.yaml in configs predating it) - where UID pins are saved.
	usersPath string
	cfgMu     sync.RWMutex

	limiterManager atomic.Pointer[auth.LimiterManager]
	rpcHandler     *rpc.RPCHandler

	sessions       = make(map[string]*rpc.Session)
	sessionsMu     sync.RWMutex
	sessionCounter int64

	requestGroup  singleflight.Group
	rpcCache      = make(map[string]rpc.CacheEntry)
	cacheMu       sync.RWMutex
	resourceCache = cache.NewTTLCache()
)

// loadConfig reads daemon.yaml and the users. At startup, unknown keys are
// only warned about, so an upgrade can't stop the daemon over a key an
// older version accepted; a reload (reloadConfig) rejects them.
func loadConfig() (Config, string, error) {
	c, users, err := config.LoadConfigDir(configDir, true)
	if err == nil {
		return c, users, nil
	}
	lenient, lusers, lerr := config.LoadConfigDir(configDir, false)
	if lerr != nil {
		return c, "", lerr
	}
	log.Printf("WARNING: %v - ignoring the unknown key(s); daemon/reload-config and linuxctl edit will refuse this file until it's fixed", err)
	return lenient, lusers, nil
}

// legacyUsersWarning is set when users still live in daemon.yaml.
func legacyUsersWarning(c Config, users string) string {
	if users != configDir+"/"+config.DaemonFile || len(c.Users) == 0 {
		return ""
	}
	return fmt.Sprintf("users are still listed in %s/%s - they now belong in %s/%s (linuxctl moves them on its next user change)", configDir, config.DaemonFile, configDir, config.UsersFile)
}
