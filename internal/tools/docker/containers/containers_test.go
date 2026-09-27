package containers

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker/fakeengine"
)

const listBody = `[
 {"Id":"aaaaaaaaaaaa1111111111","Names":["/web-1"],"Image":"nginx:alpine","State":"running","Status":"Up 2 hours",
  "Created":1700000000,"Command":"nginx -g 'daemon off;'",
  "Ports":[{"IP":"0.0.0.0","PrivatePort":80,"PublicPort":8080,"Type":"tcp"},{"PrivatePort":443,"Type":"tcp"}]},
 {"Id":"bbbbbbbbbbbb2222222222","Names":["/db-1"],"Image":"postgres:16","State":"exited","Status":"Exited (0) 1 hour ago",
  "Created":1700000001,"Ports":[]}
]`

func run(t *testing.T, e *fakeengine.Engine, args map[string]interface{}) string {
	t.Helper()
	args["_docker_socket"] = e.Socket
	b, _ := json.Marshal(args)
	out, err := List(b)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return out
}

// T1: the listing reports both containers, docker ps-style ports, and filters.
func TestList(t *testing.T) {
	e, err := fakeengine.New(map[string]string{"GET /containers/json": listBody})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out := run(t, e, map[string]interface{}{"all": true})
	for _, want := range []string{"[running] web-1 (aaaaaaaaaaaa)", "0.0.0.0:8080->80/tcp, 443/tcp", "[exited] db-1", "postgres:16"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if q, _ := e.Called("GET", "/containers/json"); !strings.Contains(q, "all=true") {
		t.Errorf("all:true did not reach the API, query was %q", q)
	}

	if out := run(t, e, map[string]interface{}{"pattern": "web-*"}); !strings.Contains(out, "web-1") || strings.Contains(out, "db-1") {
		t.Errorf("pattern did not filter:\n%s", out)
	}
	if out := run(t, e, map[string]interface{}{"state": "exited"}); !strings.Contains(out, "db-1") || strings.Contains(out, "web-1") {
		t.Errorf("state did not filter:\n%s", out)
	}
	// A state filter is meaningless without all=true, so it implies it.
	if q, _ := e.Called("GET", "/containers/json"); !strings.Contains(q, "all=true") {
		t.Errorf("state filter did not imply all=true, query was %q", q)
	}
	if out := run(t, e, map[string]interface{}{"pattern": "nothing-*"}); !strings.Contains(out, "No containers found") {
		t.Errorf("empty result should say so, got %q", out)
	}
}

func TestListJSON(t *testing.T) {
	e, err := fakeengine.New(map[string]string{"GET /containers/json": listBody})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(run(t, e, map[string]interface{}{"output_format": "json"})), &rows); err != nil {
		t.Fatalf("output_format json is not JSON: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[0]["id"] != "aaaaaaaaaaaa" || rows[0]["full_id"] != "aaaaaaaaaaaa1111111111" {
		t.Errorf("short and full id should both be reported: %v", rows[0])
	}
	if got := fmt.Sprint(rows[0]["created"]); got != "2023-11-14T22:13:20Z" {
		t.Errorf("created should be RFC3339 UTC, got %q", got)
	}
	if out := run(t, e, map[string]interface{}{"output_format": "json", "pattern": "none"}); out != "[]" {
		t.Errorf("empty structured result should be [], got %q", out)
	}
}
