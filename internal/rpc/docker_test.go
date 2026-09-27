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
	for _, unwanted := range []string{"docker/logs", "docker/images", "docker/volumes", "docker/networks"} {
		if ops[unwanted] {
			t.Errorf("%s was not granted but is listed", unwanted)
		}
	}
	if len(names("nobody")) != 0 {
		t.Errorf("a user with no docker grants should see no docker tools, saw %v", names("nobody"))
	}
}
