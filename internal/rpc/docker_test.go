package rpc

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
)

func dockerSudo(t *testing.T) *config.SudoConfig {
	t.Helper()
	c, err := config.ParseSudoConfig([]byte(`users:
  ops:
    privileged:
      tools:
        docker/containers: {allowed: true}
        docker/manage: {allowed: true, containers: ["web-*"]}
        docker/exec: {allowed: true, containers: [sandbox-1]}
        docker/prune: {allowed: true, prune: [images, build-cache]}
  nobody:
    privileged:
      tools: {}
`), true)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func prepared(t *testing.T, cfg *config.SudoConfig, user, tool, args string) map[string]interface{} {
	t.Helper()
	out, err := prepareDockerCall(cfg, user, tool, json.RawMessage(args))
	if err != nil {
		t.Fatalf("%s as %s: %v", tool, user, err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("prepared args are not JSON: %v", err)
	}
	return got
}

// T4: a docker call is root or refused - there is no unprivileged fallback,
// and the refusal says why rather than talking about `privileged: true`.
func TestPrepareDockerCallForcesRootOrRefuses(t *testing.T) {
	cfg := dockerSudo(t)
	docker.SocketPath = "/var/run/docker.sock"

	got := prepared(t, cfg, "ops", "docker/containers", `{"all":true}`)
	if got["privileged"] != true {
		t.Errorf("privileged must be forced into the args so the audit line records it: %v", got)
	}
	if got["_docker_socket"] != "/var/run/docker.sock" {
		t.Errorf("the socket from daemon.yaml must be injected (worker mode never reads it): %v", got)
	}
	if got["all"] != true {
		t.Errorf("the caller's own arguments must survive: %v", got)
	}

	if _, err := prepareDockerCall(cfg, "nobody", "docker/containers", json.RawMessage(`{}`)); err == nil {
		t.Fatal("an ungranted user must be refused")
	} else if !strings.Contains(err.Error(), "root-owned") {
		t.Errorf("the refusal should explain there is no unprivileged mode, got %v", err)
	}

	// T4 (FR-013): the read-only listings are refused the same way. Read-only
	// is not a reason to fall back to the caller's own uid - a uid outside the
	// docker group cannot open the socket at all.
	if _, err := prepareDockerCall(cfg, "ops", "docker/networks", json.RawMessage(`{}`)); err == nil {
		t.Error("docker/networks without its own grant must be refused")
	}
}

// T5b: _containers is set from this user's grant for this tool, and only for
// the tools that name a container - and a caller can never supply either
// injected key.
func TestPrepareDockerCallInjectsPerToolAllowlist(t *testing.T) {
	cfg := dockerSudo(t)
	docker.SocketPath = "/var/run/docker.sock"

	manage := prepared(t, cfg, "ops", "docker/manage", `{"container":"web-1","action":"stop"}`)
	if list, _ := json.Marshal(manage["_containers"]); string(list) != `["web-*"]` {
		t.Errorf("docker/manage should get its own list, got %s", list)
	}
	exec := prepared(t, cfg, "ops", "docker/exec", `{"container":"sandbox-1"}`)
	if list, _ := json.Marshal(exec["_containers"]); string(list) != `["sandbox-1"]` {
		t.Errorf("docker/exec must not inherit docker/manage's reach, got %s", list)
	}
	if _, set := prepared(t, cfg, "ops", "docker/containers", `{}`)["_containers"]; set {
		t.Error("a tool that names no container should get no allowlist")
	}

	// T3 (FR-012): docker/prune's allowlist is a separate one, injected the
	// same way and just as unforgeable.
	prune := prepared(t, cfg, "ops", "docker/prune", `{"_prune":["volumes"],"target":"images"}`)
	if list, _ := json.Marshal(prune["_prune"]); string(list) != `["images","build-cache"]` {
		t.Errorf("a caller-supplied prune list must be replaced by the grant's, got %s", list)
	}
	if _, set := prepared(t, cfg, "ops", "docker/manage", `{"container":"web-1"}`)["_prune"]; set {
		t.Error("only docker/prune should get a prune allowlist")
	}

	// Delete-then-set: a caller passing the injected keys cannot widen anything.
	forged := prepared(t, cfg, "ops", "docker/manage", `{"_containers":["*"],"_docker_socket":"/tmp/evil.sock","container":"web-1"}`)
	if list, _ := json.Marshal(forged["_containers"]); string(list) != `["web-*"]` {
		t.Errorf("a caller-supplied allowlist must be dropped, got %s", list)
	}
	if forged["_docker_socket"] != "/var/run/docker.sock" {
		t.Errorf("a caller-supplied socket must be dropped, got %v", forged["_docker_socket"])
	}
}

// An ungranted docker tool is absent from tools/list entirely, not listed as
// something that would fail when called.
func TestDockerToolsListingFollowsGrants(t *testing.T) {
	cfg := dockerSudo(t)

	names := func(user string) map[string]bool {
		out := map[string]bool{}
		for _, tool := range dockerTools(cfg, user) {
			out[tool.(map[string]interface{})["name"].(string)] = true
		}
		return out
	}
	ops := names("ops")
	for _, want := range []string{"docker/containers", "docker/manage", "docker/exec"} {
		if !ops[want] {
			t.Errorf("%s was granted but not listed", want)
		}
	}
	if !ops["docker/prune"] {
		t.Error("docker/prune was granted but not listed")
	}
	for _, unwanted := range []string{"docker/logs", "docker/images", "docker/volumes", "docker/networks"} {
		if ops[unwanted] {
			t.Errorf("%s was not granted but is listed", unwanted)
		}
	}
	if len(names("nobody")) != 0 {
		t.Errorf("a user with no docker grants should see no docker tools, saw %v", names("nobody"))
	}
}

// The audit line for docker/prune is the one place tool output is logged, so
// it must stay counts-and-bytes in both output shapes: the JSON one carries
// every deleted object's ID and must never reach the log verbatim.
func TestPruneSummaryLogsCountsNotObjects(t *testing.T) {
	text := "containers: 3 removed, 20.0 KiB reclaimed\n  7d1dc187b4ff\n  9a6c2228b515\n  28fa44079c42\n"
	got := pruneSummary(text)
	if strings.Contains(got, "7d1dc187b4ff") {
		t.Errorf("text summary leaked object names: %q", got)
	}
	if !strings.Contains(got, "containers: 3 removed") {
		t.Errorf("text summary lost its counts: %q", got)
	}

	js := `{"target":"volumes","deleted":["c5724212278f01cb0b2e4bb36ebc051c2af0c9ba29c652b49062c2b5fbf880cb"],"count":1,"space_reclaimed_bytes":4096}`
	got = pruneSummary(js)
	if strings.Contains(got, "c5724212278f") {
		t.Errorf("json summary leaked the deleted volume id: %q", got)
	}
	if got != "volumes: 1 removed, 4096 bytes" {
		t.Errorf("json summary = %q", got)
	}
}
