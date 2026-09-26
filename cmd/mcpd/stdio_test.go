package main

import (
	"bytes"
	"encoding/json"
	"os/user"
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"github.com/nucleusv/linux-mcp-daemon/internal/rpc"
)

func TestServeStdio(t *testing.T) {
	h := rpc.NewRPCHandler(&config.SudoConfig{}, 30, nil, &requestGroup, rpcCache, &cacheMu, resourceCache)
	in := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		``,
		`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":7}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`not json`,
		`{"jsonrpc":"2.0","id":3,"method":"no/such/method"}`,
	}, "\n") + "\n"

	var out bytes.Buffer
	serveStdio(h, "nobody", strings.NewReader(in), &out)

	byID := map[string]map[string]interface{}{}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	for _, l := range lines {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("stdout carries something that isn't one JSON-RPC message per line: %q", l)
		}
		byID[jsonID(m["id"])] = m
	}
	// initialize, tools/list, the parse error and the unknown method - and
	// nothing for the two notifications or the blank line.
	if len(lines) != 4 {
		t.Fatalf("got %d responses, want 4:\n%s", len(lines), out.String())
	}
	if r, _ := byID["1"]["result"].(map[string]interface{}); r == nil || r["serverInfo"] == nil {
		t.Errorf("initialize: no serverInfo in %v", byID["1"])
	}
	if r, _ := byID["2"]["result"].(map[string]interface{}); r == nil {
		t.Errorf("tools/list: no result in %v", byID["2"])
	} else if tools, _ := r["tools"].([]interface{}); len(tools) == 0 {
		t.Errorf("tools/list returned no tools")
	}
	if e, _ := byID["null"]["error"].(map[string]interface{}); e == nil || e["code"] != float64(-32700) {
		t.Errorf("bad JSON: want a -32700 parse error with id null, got %v", byID["null"])
	}
	if e, _ := byID["3"]["error"].(map[string]interface{}); e == nil || e["code"] != float64(-32601) {
		t.Errorf("unknown method: want -32601, got %v", byID["3"])
	}
}

func jsonID(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestStdioUser(t *testing.T) {
	self, err := user.Current()
	if err != nil {
		t.Skip("no current user")
	}
	if self.Uid == "0" {
		t.Skip("run as a non-root user")
	}
	// Not root: the workers run as whoever started mcpd.
	if got, err := stdioUser("", 1000); err != nil || got != self.Username {
		t.Errorf("stdioUser(\"\") = %q, %v; want %q", got, err, self.Username)
	}
	if _, err := stdioUser("root", 1000); err == nil {
		t.Errorf("--user another account without root: want an error")
	}
	// As root: --user is required and may not name root.
	if _, err := stdioUser("", 0); err == nil {
		t.Errorf("root without --user: want an error")
	}
	if _, err := stdioUser("root", 0); err == nil {
		t.Errorf("--user root: want an error")
	}
	if got, err := stdioUser(self.Username, 0); err != nil || got != self.Username {
		t.Errorf("root with --user %s = %q, %v", self.Username, got, err)
	}
}
