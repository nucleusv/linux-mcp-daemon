package docker

import (
	"strings"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
)

func sudo(t *testing.T, doc string) *config.SudoConfig {
	t.Helper()
	c, err := config.ParseSudoConfig([]byte(doc), true)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// T20: both grants are required, and each refusal names the one that is
// missing - there is no unprivileged fallback to drop to.
func TestBothGrantsRequired(t *testing.T) {
	noScheme := sudo(t, `users:
  ops:
    privileged:
      tools:
        docker/inspect: {allowed: true, containers: ["*"]}
`)
	if _, _, err := Handle("docker-container://web-1/status", "ops", noScheme); err == nil || !strings.Contains(err.Error(), `"docker-container://"`) {
		t.Errorf("a missing scheme grant should name the scheme, got %v", err)
	}

	noWorker := sudo(t, `users:
  ops:
    privileged:
      tools: {}
      resources:
        # A whole-scheme grant is the empty prefix, as in the shipped config;
        # a literal "*" would grant only a container actually named "*".
        "docker-container://": [""]
`)
	if _, _, err := Handle("docker-container://web-1/status", "ops", noWorker); err == nil || !strings.Contains(err.Error(), "docker/inspect") {
		t.Errorf("a missing worker grant should name docker/inspect, got %v", err)
	}
}

// T21: one worker behind three schemes - the URI is parsed here, and the parse
// rule differs per scheme because an image reference contains slashes and a
// container name does not.
func TestURIParsing(t *testing.T) {
	// Refused at the scheme grant, which is after parsing: an unknown view or
	// an unknown scheme must fail with its own message, not that one.
	cfg := sudo(t, `users:
  ops:
    privileged:
      tools: {}
`)
	for _, tc := range []struct{ uri, want string }{
		{"docker-container://web-1/history", "unknown view"},
		{"network://bridge/inspect", "unknown resource"},
		{"docker-container://", "no name"},
		{"container:/web-1", "unknown resource"},
	} {
		if _, _, err := Handle(tc.uri, "ops", cfg); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: want %q, got %v", tc.uri, tc.want, err)
		}
	}

	// A valid URI gets as far as the grant check, whose message quotes the
	// scheme - which is how these assert what each URI parsed to without
	// spawning a worker.
	for _, tc := range []struct{ uri, scheme string }{
		{"docker-container://web-1/status", "docker-container://"},
		{"docker-container://web-1", "docker-container://"},
		{"docker-image://ghcr.io/org/api:v1/inspect", "docker-image://"},
		{"docker-volume://pgdata/inspect", "docker-volume://"},
		{"docker-network://bridge/inspect", "docker-network://"},
	} {
		_, _, err := Handle(tc.uri, "ops", cfg)
		if err == nil || !strings.Contains(err.Error(), tc.scheme) {
			t.Errorf("%s should parse and then need the %s grant, got %v", tc.uri, tc.scheme, err)
		}
	}
}

// T3: docker-network:// and the host's network:// are separate namespaces, and
// that is the whole reason for the prefix. network://... never reaches this
// handler, and a Docker network *named* "interfaces" or "routes" - which would
// be unreadable if Docker networks shared the bare scheme - is reachable, with
// its own name intact.
func TestNoSchemeCollisionWithHostNetworking(t *testing.T) {
	cfg := sudo(t, `users:
  ops:
    privileged:
      tools: {}
`)
	for _, uri := range []string{"network://interfaces", "network://routes", "network://interfaces/eth0", "network://bridge/inspect"} {
		if _, _, err := Handle(uri, "ops", cfg); err == nil || !strings.Contains(err.Error(), "unknown resource") {
			t.Errorf("%s belongs to host networking, not here, got %v", uri, err)
		}
	}
	// Refused at the grant check, which only happens once the URI parsed - so
	// reaching that message is the assertion that the name survived the parse.
	for _, name := range []string{"interfaces", "routes", "bridge"} {
		_, _, err := Handle("docker-network://"+name+"/inspect", "ops", cfg)
		if err == nil || !strings.Contains(err.Error(), `"docker-network://"`) || !strings.Contains(err.Error(), name) {
			t.Errorf("docker-network://%s should parse to that name, got %v", name, err)
		}
	}
}

// The scheme grant is path-scoped like every other resource grant: a narrow
// list does not cover a container outside it.
func TestSchemeGrantIsScoped(t *testing.T) {
	cfg := sudo(t, `users:
  ops:
    privileged:
      tools:
        docker/inspect: {allowed: true, containers: ["web-*"]}
      resources:
        "docker-container://": ["web-1"]
`)
	if _, _, err := Handle("docker-container://db-1/status", "ops", cfg); err == nil || !strings.Contains(err.Error(), `"docker-container://"`) {
		t.Errorf("a container outside the resources: list should be refused, got %v", err)
	}
}
