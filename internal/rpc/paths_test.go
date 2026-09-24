package rpc

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/cache"
	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"golang.org/x/sync/singleflight"
)

func callTool(t *testing.T, h *RPCHandler, user, name, args string) (string, bool) {
	t.Helper()
	params, _ := json.Marshal(map[string]interface{}{"name": name, "arguments": json.RawMessage(args)})
	var resp JSONRPCResponse
	h.HandleToolsCall(&Session{User: user}, JSONRPCRequest{Params: params}, &resp)
	res, ok := resp.Result.(ToolResult)
	if !ok || len(res.Content) == 0 {
		t.Fatalf("%s: unexpected result %#v", name, resp.Result)
	}
	return res.Content[0].Text, res.IsError
}

// paths: must limit every tool that takes a path when run as root - not
// only files/*. Denied calls never reach a worker.
func TestPrivilegedPathLimits(t *testing.T) {
	sudo, err := config.ParseSudoConfig([]byte(`users:
  alice:
    privileged:
      tools:
        disks/usage: {allowed: true, paths: [/var]}
        disks/free: {allowed: true}
        files/list: {allowed: true}
`), true)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.RWMutex
	h := NewRPCHandler(sudo, 5, nil, &singleflight.Group{}, map[string]CacheEntry{}, &mu, cache.NewTTLCache())

	denied := []struct{ tool, args string }{
		{"disks/usage", `{"path": "/root", "privileged": true}`},
		{"disks/usage", `{"path": "/var/../root", "privileged": true}`},
		{"disks/usage", `{"path": "/varnish", "privileged": true}`},
		{"files/list", `{"path": "/tmp", "privileged": true}`}, // files/*: no paths = no root
	}
	for _, c := range denied {
		text, isErr := callTool(t, h, "alice", c.tool, c.args)
		if !isErr || !strings.Contains(text, "not authorized to run "+c.tool+" on path") {
			t.Errorf("%s %s: want a path denial, got %q", c.tool, c.args, text)
		}
	}

	// Inside the grant's paths, or a tool granted without paths: the path
	// check passes (the call then fails later, at the worker, in a test).
	allowed := []struct{ tool, args string }{
		{"disks/usage", `{"path": "/var/log", "privileged": true}`},
		{"disks/free", `{"path": "/root", "privileged": true}`},
		{"disks/usage", `{"path": "/root"}`}, // not as root: no limit
	}
	for _, c := range allowed {
		if text, _ := callTool(t, h, "alice", c.tool, c.args); strings.Contains(text, "on path") {
			t.Errorf("%s %s: unexpectedly denied: %q", c.tool, c.args, text)
		}
	}
}
