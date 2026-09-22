package rpc

import (
	"sync"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/cache"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
	"golang.org/x/sync/singleflight"
)

// RPCHandler contains the dependencies required by the RPC endpoints.
type RPCHandler struct {
	SudoConfig       *config.SudoConfig
	WorkerTimeoutSec int
	ToolsConfig      map[string]struct {
		TimeoutSeconds int `yaml:"timeout_seconds"`
	}
	RequestGroup  *singleflight.Group
	RpcCache      map[string]CacheEntry
	CacheMu       *sync.RWMutex
	ResourceCache *cache.TTLCache
}

// NewRPCHandler creates a new RPCHandler with its dependencies.
func NewRPCHandler(
	sudoConfig *config.SudoConfig,
	workerTimeoutSec int,
	toolsConfig map[string]struct {
		TimeoutSeconds int `yaml:"timeout_seconds"`
	},
	requestGroup *singleflight.Group,
	rpcCache map[string]CacheEntry,
	cacheMu *sync.RWMutex,
	resourceCache *cache.TTLCache,
) *RPCHandler {
	return &RPCHandler{
		SudoConfig:       sudoConfig,
		WorkerTimeoutSec: workerTimeoutSec,
		ToolsConfig:      toolsConfig,
		RequestGroup:     requestGroup,
		RpcCache:         rpcCache,
		CacheMu:          cacheMu,
		ResourceCache:    resourceCache,
	}
}
