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
