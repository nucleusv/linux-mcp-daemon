package volumes

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker/fakeengine"
)

const (
	volBody = `{"Volumes":[
 {"Name":"pgdata","Driver":"local","Mountpoint":"/var/lib/docker/volumes/pgdata/_data","Scope":"local","CreatedAt":"2026-09-01T10:00:00Z"},
 {"Name":"cache","Driver":"local","Mountpoint":"/var/lib/docker/volumes/cache/_data","Scope":"local","CreatedAt":"2026-09-02T10:00:00Z"}
],"Warnings":[]}`
	ctBody = `[
 {"Names":["/db-1"],"Mounts":[{"Type":"volume","Name":"pgdata"},{"Type":"bind","Name":""}]},
 {"Names":["/api-1"],"Mounts":[{"Type":"volume","Name":"pgdata"}]}
]`
)

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

// T15: volumes are listed with the containers mounting them - the names, not
// Docker's bare refcount - and an unused volume says so.
func TestList(t *testing.T) {
	e, err := fakeengine.New(map[string]string{"GET /volumes": volBody, "GET /containers/json": ctBody})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out := run(t, e, map[string]interface{}{})
	for _, want := range []string{"pgdata", "In use by: api-1, db-1", "cache", "In use by: (nothing)", "/var/lib/docker/volumes/pgdata/_data"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if q, _ := e.Called("GET", "/containers/json"); !strings.Contains(q, "all=true") {
		t.Errorf("mounters must be looked up across stopped containers too, query was %q", q)
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(run(t, e, map[string]interface{}{"output_format": "json", "pattern": "pg*"})), &rows); err != nil {
		t.Fatalf("output_format json is not JSON: %v", err)
	}
	if len(rows) != 1 || rows[0]["name"] != "pgdata" {
		t.Fatalf("pattern did not filter: %v", rows)
	}
	users, _ := json.Marshal(rows[0]["in_use_by"])
	if string(users) != `["api-1","db-1"]` {
		t.Errorf("in_use_by should name both mounters, got %s", users)
	}
}

// A container list that fails must not fail the volume listing: the names are
// an enrichment, the volumes are the answer.
func TestListSurvivesContainerListFailure(t *testing.T) {
	e, err := fakeengine.New(map[string]string{"GET /volumes": volBody})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	if out := run(t, e, map[string]interface{}{}); !strings.Contains(out, "pgdata") {
		t.Errorf("volumes should still be listed:\n%s", out)
	}
}
