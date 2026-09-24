package rpc

import (
	"sync"
	"sync/atomic"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/cache"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
	"golang.org/x/sync/singleflight"
)

// ToolsConfig holds per-tool overrides from daemon.yaml's tools: block.
type ToolsConfig = config.ToolsConfig

// RPCHandler contains the dependencies required by the RPC endpoints.
type RPCHandler struct {
	// settings is swapped as a whole by Reconfigure (daemon/reload-config),
	// so a request always sees one consistent config, old or new.
	settings atomic.Pointer[settings]

	// ReloadConfig re-reads daemon.yaml and mcp-sudo.yaml and applies them
	// (set by cmd/mcpd, which owns daemon.yaml). It returns a summary of
	// what changed.
	ReloadConfig func(byUser string) (string, error)

	RequestGroup  *singleflight.Group
	RpcCache      map[string]CacheEntry
	CacheMu       *sync.RWMutex
	ResourceCache *cache.TTLCache
}

type settings struct {
	sudo             *config.SudoConfig
	workerTimeoutSec int
	tools            ToolsConfig
}

// NewRPCHandler creates a new RPCHandler with its dependencies.
func NewRPCHandler(
	sudoConfig *config.SudoConfig,
	workerTimeoutSec int,
	toolsConfig ToolsConfig,
	requestGroup *singleflight.Group,
	rpcCache map[string]CacheEntry,
	cacheMu *sync.RWMutex,
	resourceCache *cache.TTLCache,
) *RPCHandler {
	h := &RPCHandler{
		RequestGroup:  requestGroup,
		RpcCache:      rpcCache,
		CacheMu:       cacheMu,
		ResourceCache: resourceCache,
	}
	h.Reconfigure(sudoConfig, workerTimeoutSec, toolsConfig)
	return h
}

// Reconfigure swaps in a new config and drops every cached result, since
// cached output may have been produced under grants that no longer exist.
func (h *RPCHandler) Reconfigure(sudoConfig *config.SudoConfig, workerTimeoutSec int, toolsConfig ToolsConfig) {
	h.settings.Store(&settings{sudo: sudoConfig, workerTimeoutSec: workerTimeoutSec, tools: toolsConfig})
	if h.CacheMu != nil {
		h.CacheMu.Lock()
		for k := range h.RpcCache {
			delete(h.RpcCache, k)
		}
		h.CacheMu.Unlock()
	}
	if h.ResourceCache != nil {
		h.ResourceCache.Clear()
	}
}

// Sudo returns the mcp-sudo.yaml rules currently in effect.
func (h *RPCHandler) Sudo() *config.SudoConfig { return h.settings.Load().sudo }

// timeoutFor returns the worker timeout for a tool: its own override, or
// the global worker default.
func (h *RPCHandler) timeoutFor(tool string) int {
	s := h.settings.Load()
	if t, ok := s.tools[tool]; ok && t.TimeoutSeconds > 0 {
		return t.TimeoutSeconds
	}
	return s.workerTimeoutSec
}
