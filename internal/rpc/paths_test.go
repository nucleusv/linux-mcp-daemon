package rpc

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/cache"
	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"github.com/nucleusv/linux-mcp-daemon/internal/logging"
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
        disks/free: {allowed: true, paths: ["/"]}
        files/list: {allowed: true, paths: [/srv]}
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
		{"files/list", `{"path": "/tmp", "privileged": true}`}, // outside its paths
	}
	for _, c := range denied {
		text, isErr := callTool(t, h, "alice", c.tool, c.args)
		if !isErr || !strings.Contains(text, "not authorized to run "+c.tool+" on path") {
			t.Errorf("%s %s: want a path denial, got %q", c.tool, c.args, text)
		}
	}

	// Inside the grant's paths (paths: ["/"] covers everything): the path
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

// One line per tool call at info; the denied call at warn; mutating calls
// as audit lines even at error level; secrets in arguments never appear.
func TestToolCallLogging(t *testing.T) {
	sudo, _ := config.ParseSudoConfig([]byte("users:\n  alice:\n    privileged:\n      tools: {}\n"), true)
	var mu sync.RWMutex
	h := NewRPCHandler(sudo, 5, nil, &singleflight.Group{}, map[string]CacheEntry{}, &mu, cache.NewTTLCache())

	var buf bytes.Buffer
	logging.SetOutput(&buf, logging.Config{Level: "error"})
	defer logging.SetOutput(io.Discard, logging.Config{})

	callTool(t, h, "alice", "files/list", `{"path": "/root", "privileged": true}`)                                   // denied, warn: hidden at error
	callTool(t, h, "alice", "files/create", `{"path": "/tmp/x", "content": "SECRET-FILE-BODY", "privileged": true}`) // mutating: audit
	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 || !strings.Contains(out, "audit=true") || !strings.Contains(out, "tool=files/create") {
		t.Errorf("at error level want exactly the audit line, got:\n%s", out)
	}
	if strings.Contains(out, "SECRET-FILE-BODY") {
		t.Errorf("file content leaked into the log:\n%s", out)
	}

	buf.Reset()
	logging.Configure(logging.Config{Level: "info"})
	callTool(t, h, "alice", "files/list", `{"path": "/root", "privileged": true}`)
	if !strings.Contains(buf.String(), " WARN  tool call denied ") {
		t.Errorf("denied call not logged at warn:\n%s", buf.String())
	}
}

func TestNoFollowHelpers(t *testing.T) {
	for _, c := range []struct {
		allowed []string
		want    bool
	}{
		{[]string{"/"}, true}, {[]string{"/tmp", "//"}, true}, {[]string{"/tmp"}, false}, {nil, false}, {[]string{"relative"}, false},
	} {
		if got := config.CoversRoot(c.allowed); got != c.want {
			t.Errorf("CoversRoot(%v) = %t", c.allowed, got)
		}
	}
	got := string(withoutKey(json.RawMessage(`{"path":"/tmp","_no_follow":false}`), "_no_follow"))
	if strings.Contains(got, "_no_follow") || !strings.Contains(got, `"path":"/tmp"`) {
		t.Errorf("withoutKey: %s", got)
	}
}

func TestRelativePathRejected(t *testing.T) {
	sudo, _ := config.ParseSudoConfig([]byte("users:\n  alice:\n    privileged:\n      tools:\n        files/list: {allowed: true, paths: [\"/\"]}\n"), true)
	var mu sync.RWMutex
	h := NewRPCHandler(sudo, 5, nil, &singleflight.Group{}, map[string]CacheEntry{}, &mu, cache.NewTTLCache())
	for _, args := range []string{`{"path": "tmp/x"}`, `{"path": "tmp/x", "privileged": true}`} {
		text, isErr := callTool(t, h, "alice", "files/list", args)
		if !isErr || !strings.Contains(text, `path must be absolute, got "tmp/x"`) {
			t.Errorf("%s: got %q", args, text)
		}
	}
}
