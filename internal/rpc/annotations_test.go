package rpc

import (
	"sync"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/cache"
	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"golang.org/x/sync/singleflight"
)

// Every tool tools/list returns carries annotations, and every annotation row
// belongs to a tool that exists (FR-024). A new tool without a row fails here.
func TestEveryListedToolHasConsistentAnnotations(t *testing.T) {
	sudo, err := config.ParseSudoConfig([]byte("users: {}\n"), true)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.RWMutex
	h := NewRPCHandler(sudo, 5, nil, &singleflight.Group{}, map[string]CacheEntry{}, &mu, cache.NewTTLCache())
	var resp JSONRPCResponse
	h.HandleToolsList(&Session{User: "nobody"}, &resp)

	listed := map[string]bool{}
	for _, raw := range resp.Result.(map[string]interface{})["tools"].([]interface{}) {
		tool := raw.(map[string]interface{})
		name := tool["name"].(string)
		listed[name] = true
		ann, ok := tool["annotations"].(map[string]interface{})
		if !ok {
			t.Errorf("%s has no annotations - add a row to toolAnnotations", name)
			continue
		}
		for _, k := range []string{"title", "readOnlyHint", "destructiveHint", "idempotentHint", "openWorldHint"} {
			if _, has := ann[k]; !has {
				t.Errorf("%s: annotations lack %s", name, k)
			}
		}
		if ann["title"] == "" {
			t.Errorf("%s: empty title", name)
		}
		if ann["readOnlyHint"] == true && (ann["destructiveHint"] == true || ann["idempotentHint"] != true) {
			t.Errorf("%s: a read-only tool must be non-destructive and idempotent, got %v", name, ann)
		}
	}
	for name := range toolAnnotations {
		if !listed[name] {
			t.Errorf("annotation row for %s, which tools/list does not return", name)
		}
	}
}
