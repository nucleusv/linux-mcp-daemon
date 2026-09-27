package images

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker/fakeengine"
)

const body = `[
 {"Id":"sha256:1111111111112222","RepoTags":["nginx:alpine"],"Created":1700000000,"Size":11534336,"Containers":1},
 {"Id":"sha256:3333333333334444","RepoTags":[],"RepoDigests":["postgres@sha256:dead"],"Created":1700000001,"Size":429496729,"Containers":0}
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

func TestList(t *testing.T) {
	e, err := fakeengine.New(map[string]string{"GET /images/json": body})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out := run(t, e, map[string]interface{}{"all": true})
	for _, want := range []string{"nginx:alpine", "ID: 111111111111", "11.0 MiB", "<none>:<none>", "409.6 MiB"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if q, _ := e.Called("GET", "/images/json"); q != "all=true" {
		t.Errorf("all:true did not reach the API, query was %q", q)
	}

	// A pattern without a tag still matches a tagged image.
	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(run(t, e, map[string]interface{}{"output_format": "json", "pattern": "nginx"})), &rows); err != nil {
		t.Fatalf("output_format json is not JSON: %v", err)
	}
	if len(rows) != 1 || rows[0]["id"] != "111111111111" {
		t.Fatalf("pattern should match the repository alone: %v", rows)
	}
	if rows[0]["full_id"] != "sha256:1111111111112222" {
		t.Errorf("full_id should keep Docker's digest form: %v", rows[0]["full_id"])
	}
}
