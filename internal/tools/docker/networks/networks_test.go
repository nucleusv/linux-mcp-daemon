package networks

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker/fakeengine"
)

// Three networks: the default bridge, an internal user-defined one, and host
// (no IPAM config, no flags).
//
// Every Containers map is empty on purpose - that is what a real Docker answers
// on GET /networks, however many containers are attached, and believing
// otherwise is what made the first version of this tool print "Attached:
// (nothing)" for a network with two containers on it.
const netBody = `[
 {"Name":"bridge","Id":"f1b2c3d4e5f60718293a4b5c6d7e8f901a2b3c4d5e6f708192a3b4c5d6e7f801","Created":"2026-09-01T10:00:00.1Z",
  "Scope":"local","Driver":"bridge","EnableIPv6":false,"Internal":false,"Attachable":false,"Ingress":false,
  "IPAM":{"Driver":"default","Config":[{"Subnet":"172.17.0.0/16","Gateway":"172.17.0.1"}]},
  "Options":{"com.docker.network.bridge.default_bridge":"true"},"Labels":{},"Containers":{}},
 {"Name":"backend","Id":"aa11bb22cc33dd44ee55ff6677889900aabbccddeeff00112233445566778899","Created":"2026-09-02T10:00:00.1Z",
  "Scope":"local","Driver":"bridge","EnableIPv6":true,"Internal":true,"Attachable":true,"Ingress":false,
  "IPAM":{"Driver":"default","Config":[{"Subnet":"10.5.0.0/24","Gateway":"10.5.0.1"}]},
  "Options":{},"Labels":{},"Containers":{}},
 {"Name":"host","Id":"cc99dd88ee77ff66aa55bb44cc33dd22ee11ff0099887766554433221100aabb","Created":"2026-09-01T09:00:00.1Z",
  "Scope":"local","Driver":"host","IPAM":{"Driver":"default","Config":[]},"Containers":{}}
]`

// web-1 and db-1 are on bridge, db-1 is also stopped-and-on-backend with no
// address - the membership the listing prints comes from here.
const containerBody = `[
 {"Id":"e36aeba4369a","Names":["/web-1"],"State":"running",
  "NetworkSettings":{"Networks":{"bridge":{"IPAddress":"172.17.0.2","MacAddress":"02:42:ac:11:00:02"}}}},
 {"Id":"80d3cf9c8b3c","Names":["/db-1"],"State":"exited",
  "NetworkSettings":{"Networks":{"bridge":{"IPAddress":"172.17.0.3"},"backend":{"IPAddress":""}}}}
]`

func engine(t *testing.T) *fakeengine.Engine {
	t.Helper()
	e, err := fakeengine.New(map[string]string{
		"GET /networks":        netBody,
		"GET /containers/json": containerBody,
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

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

// T1: networks listed with driver, scope, subnet, gateway and the containers on
// them - and a network with nothing attached says so rather than going silent.
func TestList(t *testing.T) {
	e := engine(t)
	defer e.Close()

	out := run(t, e, map[string]interface{}{})
	for _, want := range []string{
		"bridge (f1b2c3d4e5f6)",
		"Driver: bridge | Scope: local",
		"Subnet: 172.17.0.0/16 | Gateway: 172.17.0.1",
		"Attached: db-1 (172.17.0.3), web-1 (172.17.0.2)",
		"backend",
		"internal, attachable, ipv6",
		// db-1 is on backend with no address - a name alone, not a blank pair.
		"Attached: db-1\n",
		// host has nobody on it, and says so rather than going silent.
		"Attached: (nothing)",
		"docker-network://<name>/inspect",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// The host network sets none of the flags, so none should be printed.
	if strings.Contains(out, "internal, attachable, ipv6, ingress") {
		t.Errorf("flags must report only what is set:\n%s", out)
	}
}

// Filters: a glob on the name and an exact driver, plus the structured shape
// the JSON form promises.
func TestFilters(t *testing.T) {
	e := engine(t)
	defer e.Close()

	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(run(t, e, map[string]interface{}{"output_format": "json", "pattern": "b*"})), &rows); err != nil {
		t.Fatalf("output_format json is not JSON: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("pattern b* should match bridge and backend, got %d: %v", len(rows), rows)
	}
	if rows[0]["id"] != "f1b2c3d4e5f6" || rows[0]["full_id"] != "f1b2c3d4e5f60718293a4b5c6d7e8f901a2b3c4d5e6f708192a3b4c5d6e7f801" {
		t.Errorf("both the short and full id belong in the row: %v", rows[0])
	}
	subnets, _ := json.Marshal(rows[0]["subnets"])
	if string(subnets) != `["172.17.0.0/16"]` {
		t.Errorf("subnets: %s", subnets)
	}

	rows = nil
	if err := json.Unmarshal([]byte(run(t, e, map[string]interface{}{"output_format": "json", "driver": "HOST"})), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["name"] != "host" {
		t.Fatalf("driver filter is case-insensitive and exact, got %v", rows)
	}

	if out := run(t, e, map[string]interface{}{"pattern": "nothing-*"}); !strings.Contains(out, "No networks found") {
		t.Errorf("an empty match should say so: %q", out)
	}
}

// T5: read-only by construction. The tool must reach the socket with GET only -
// nothing here can create, remove, connect or disconnect a network.
func TestOnlyReads(t *testing.T) {
	e := engine(t)
	defer e.Close()

	run(t, e, map[string]interface{}{})
	for _, r := range e.Requests() {
		if r.Method != "GET" {
			t.Errorf("docker/networks must only read, saw %s %s", r.Method, r.Path)
		}
	}
}
