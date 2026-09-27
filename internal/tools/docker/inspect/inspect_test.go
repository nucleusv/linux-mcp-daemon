package inspect

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker/fakeengine"
)

const fullID = "aaaaaaaaaaaa1111111111"

func call(e *fakeengine.Engine, args map[string]interface{}) (string, error) {
	args["_docker_socket"] = e.Socket
	b, _ := json.Marshal(args)
	return Inspect(b)
}

func inspectBody(started string, memory int64) string {
	return `{"Id":"` + fullID + `","Name":"/web-1","Created":"2026-09-01T10:00:00Z","RestartCount":2,
 "State":{"Status":"running","Running":true,"Pid":4242,"ExitCode":0,"StartedAt":"` + started + `",
   "Health":{"Status":"healthy","FailingStreak":0}},
 "Config":{"Image":"nginx:alpine"},
 "HostConfig":{"Memory":` + itoa(memory) + `,"NanoCpus":1500000000,"Privileged":false,"RestartPolicy":{"Name":"unless-stopped"}},
 "NetworkSettings":{"Ports":{"80/tcp":[{"HostIp":"0.0.0.0","HostPort":"8080"}],"443/tcp":[]},
   "Networks":{"bridge":{"IPAddress":"172.17.0.2"}}}}`
}

func itoa(n int64) string { b, _ := json.Marshal(n); return string(b) }

// T16: the status view is computed - state, health, restart count, uptime from
// StartedAt - and reports only the limits that are actually set.
func TestStatusIsComputed(t *testing.T) {
	started := time.Now().Add(-90 * time.Second).UTC().Format(time.RFC3339Nano)
	e, err := fakeengine.New(map[string]string{
		"GET /containers/web-1/json":          fakeengine.Inspect(fullID, "web-1"),
		"GET /containers/" + fullID + "/json": inspectBody(started, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out, err := call(e, map[string]interface{}{"_containers": []string{"web-*"}, "kind": "container", "name": "web-1", "view": "status"})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("status is not JSON: %v", err)
	}
	if got["name"] != "web-1" || got["state"] != "running" || got["health"] != "healthy" || got["restart_count"] != float64(2) {
		t.Errorf("unexpected status: %v", got)
	}
	if u, _ := got["uptime"].(string); u != "1m30s" {
		t.Errorf("uptime should be computed from StartedAt, got %q", u)
	}
	limits, _ := got["limits"].(map[string]interface{})
	if limits["cpus"] != 1.5 {
		t.Errorf("NanoCpus should be reported as cpus, got %v", got["limits"])
	}
	if _, set := limits["memory_bytes"]; set {
		t.Errorf("an unset memory limit should be absent, not zero: %v", limits)
	}
	var ports []string
	for _, p := range got["ports"].([]interface{}) {
		ports = append(ports, p.(string))
	}
	sort.Strings(ports)
	if strings.Join(ports, " ") != "0.0.0.0:8080->80/tcp 443/tcp" {
		t.Errorf("published and merely exposed ports should both show, got %v", ports)
	}

	// With a limit set it appears; the key's presence is the signal.
	e2, err := fakeengine.New(map[string]string{
		"GET /containers/web-1/json":          fakeengine.Inspect(fullID, "web-1"),
		"GET /containers/" + fullID + "/json": inspectBody(started, 536870912),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e2.Close()
	out, err = call(e2, map[string]interface{}{"_containers": []string{"*"}, "kind": "container", "name": "web-1", "view": "status"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"memory_bytes": 536870912`) {
		t.Errorf("a set memory limit should be reported: %s", out)
	}
}

// T17/T18: stats is one snapshot, never a stream; top is rendered as a table.
func TestStatsAndTop(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"GET /containers/web-1/json":           fakeengine.Inspect(fullID, "web-1"),
		"GET /containers/" + fullID + "/stats": `{"cpu_stats":{"cpu_usage":{"total_usage":1}}}`,
		"GET /containers/" + fullID + "/top":   `{"Titles":["PID","USER","COMMAND"],"Processes":[["1","root","nginx"],["7","nginx","worker"]]}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	if _, err := call(e, map[string]interface{}{"_containers": []string{"*"}, "kind": "container", "name": "web-1", "view": "stats"}); err != nil {
		t.Fatalf("stats: %v", err)
	}
	q, _ := e.Called("GET", "/containers/"+fullID+"/stats")
	if q != "stream=false" {
		t.Errorf("stats must be a single snapshot (stream=false), query was %q", q)
	}

	out, err := call(e, map[string]interface{}{"_containers": []string{"*"}, "kind": "container", "name": "web-1", "view": "top"})
	if err != nil {
		t.Fatalf("top: %v", err)
	}
	var top struct {
		Titles    []string            `json:"titles"`
		Processes []map[string]string `json:"processes"`
	}
	if err := json.Unmarshal([]byte(out), &top); err != nil {
		t.Fatalf("top is not JSON: %v", err)
	}
	if len(top.Processes) != 2 || top.Processes[0]["command"] != "nginx" || top.Processes[1]["pid"] != "7" {
		t.Errorf("top rows should be keyed by lowercased title, got %v", top.Processes)
	}

	if _, err := call(e, map[string]interface{}{"_containers": []string{"*"}, "kind": "container", "name": "web-1", "view": "nonsense"}); err == nil || !strings.Contains(err.Error(), "unknown container view") {
		t.Errorf("an unknown view should be refused, got %v", err)
	}
}

// T19: image, volume and network reads pass Docker's own JSON through, and
// each identifier is validated for its own charset.
func TestOtherKinds(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"GET /images/ghcr.io/org/api:v1/json": `{"Id":"sha256:beef","RepoTags":["ghcr.io/org/api:v1"]}`,
		"GET /volumes/pgdata":                 `{"Name":"pgdata","Driver":"local"}`,
		"GET /networks/bridge":                `{"Name":"bridge","Driver":"bridge"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	// The fields are Docker's own, and the answer is indented like every other
	// resource template's - Docker sends one long line.
	for _, tc := range []struct{ kind, name, want string }{
		{"image", "ghcr.io/org/api:v1", `"RepoTags": [`},
		{"volume", "pgdata", `"Name": "pgdata"`},
		{"network", "bridge", `"Driver": "bridge"`},
	} {
		out, err := call(e, map[string]interface{}{"kind": tc.kind, "name": tc.name})
		if err != nil {
			t.Fatalf("%s: %v", tc.kind, err)
		}
		if !strings.Contains(out, tc.want) || !strings.Contains(out, "\n  ") {
			t.Errorf("%s should pass Docker's JSON through, indented, got %s", tc.kind, out)
		}
	}

	// A volume name is a container-style identifier: a slash is refused, not
	// cleaned, so it can never address another endpoint.
	if _, err := call(e, map[string]interface{}{"kind": "volume", "name": "pgdata/../../images/json"}); err == nil || !strings.Contains(err.Error(), "refused rather than cleaned") {
		t.Errorf("a slashed volume name should be refused, got %v", err)
	}
	if _, err := call(e, map[string]interface{}{"kind": "image", "name": "org/../../secret"}); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Errorf("a .. image reference should be refused, got %v", err)
	}
	if _, err := call(e, map[string]interface{}{"kind": "cluster", "name": "x"}); err == nil || !strings.Contains(err.Error(), "unknown kind") {
		t.Errorf("an unknown kind should be refused, got %v", err)
	}
}

// T2 (FR-013): a network inspect is what docker-network://{name}/inspect
// returns - IPAM and the per-container addresses the listing summarises, which
// is the reason the template exists next to docker/networks.
func TestNetworkInspect(t *testing.T) {
	e, err := fakeengine.New(map[string]string{
		"GET /networks/backend": `{"Name":"backend","Id":"aa11bb22cc33","Driver":"bridge","Scope":"local","Internal":true,` +
			`"IPAM":{"Driver":"default","Config":[{"Subnet":"10.5.0.0/24","Gateway":"10.5.0.1"}]},` +
			`"Containers":{"80d3cf9c8b3c":{"Name":"db-1","IPv4Address":"10.5.0.2/24","MacAddress":"02:42:0a:05:00:02"}},` +
			`"Options":{"com.docker.network.driver.mtu":"1500"},"Labels":{"app":"demo"}}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out, err := call(e, map[string]interface{}{"kind": "network", "name": "backend"})
	if err != nil {
		t.Fatalf("network inspect: %v", err)
	}
	for _, want := range []string{`"Subnet": "10.5.0.0/24"`, `"Gateway": "10.5.0.1"`, `"Name": "db-1"`,
		`"IPv4Address": "10.5.0.2/24"`, `"MacAddress": "02:42:0a:05:00:02"`, `"Internal": true`, `"com.docker.network.driver.mtu"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// T5: read-only. A network inspect must reach the socket with GET only.
	for _, r := range e.Requests() {
		if r.Method != "GET" {
			t.Errorf("a network inspect must only read, saw %s %s", r.Method, r.Path)
		}
	}
	// A slashed name is refused, not cleaned - it must not address /networks/prune.
	if _, err := call(e, map[string]interface{}{"kind": "network", "name": "backend/../prune"}); err == nil || !strings.Contains(err.Error(), "refused rather than cleaned") {
		t.Errorf("a slashed network name should be refused, got %v", err)
	}
}
