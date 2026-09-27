package config

import (
	"strings"
	"testing"
)

// T6: a docker tool that names a container is refused at strict parse when
// its grant has no containers: list - as root that grant could only ever be
// refused at runtime, so it is a config error, not a silent no-op.
func TestContainersRequired(t *testing.T) {
	for tool := range ContainerTools {
		doc := "users:\n  a:\n    privileged:\n      tools:\n        " + tool + ": {allowed: true}\n"
		if _, err := ParseSudoConfig([]byte(doc), true); err == nil || !strings.Contains(err.Error(), "allowed without containers") {
			t.Errorf("%s allowed without containers: err = %v", tool, err)
		}
		// T6b: the same grant must still load at startup, inert - a running
		// daemon does not refuse to boot over it.
		c, err := ParseSudoConfig([]byte(doc), false)
		if err != nil {
			t.Fatalf("%s: lenient (startup) must still load: %v", tool, err)
		}
		if got := c.GetAllowedContainers("a", tool); len(got) != 0 {
			t.Errorf("%s: an inert grant must expose no containers, got %v", tool, got)
		}
	}
}

// T5c: containers: on a docker tool that names no container has no effect, so
// strict parse rejects it rather than letting it read like a restriction.
func TestContainersOnlyOnContainerTools(t *testing.T) {
	for _, tool := range []string{"docker/containers", "docker/images", "docker/volumes"} {
		doc := "users:\n  a:\n    privileged:\n      tools:\n        " + tool + ": {allowed: true, containers: [web-1]}\n"
		if _, err := ParseSudoConfig([]byte(doc), true); err == nil || !strings.Contains(err.Error(), "containers has no effect") {
			t.Errorf("%s: containers accepted, err = %v", tool, err)
		}
		if _, err := ParseSudoConfig([]byte(doc), false); err != nil {
			t.Errorf("%s: lenient (startup) must still load: %v", tool, err)
		}
	}
}

// T5b: each container-scoped tool carries its own list. docker/exec's reach is
// never widened by what docker/manage was granted.
func TestPerToolContainerLists(t *testing.T) {
	c, err := ParseSudoConfig([]byte(`users:
  a:
    privileged:
      tools:
        docker/manage: {allowed: true, containers: ["*"]}
        docker/exec: {allowed: true, containers: [sandbox-1]}
        docker/logs: {allowed: false}
`), true)
	if err != nil {
		t.Fatal(err)
	}
	for tool, want := range map[string]string{
		"docker/manage": "*",
		"docker/exec":   "sandbox-1",
		"docker/logs":   "",
	} {
		got := strings.Join(c.GetAllowedContainers("a", tool), ",")
		if got != want {
			t.Errorf("%s: got %q, want %q", tool, got, want)
		}
	}
}

// T6c: every containers: grant in the shipped configs is explicit - "*" is
// spelled out where it is meant, so no reader has to guess whether an absent
// list means "all" (it means none).
func TestShippedDockerGrantsAreExplicit(t *testing.T) {
	for _, dir := range []string{"../../configs", "../../packaging/configs", "../../packaging/configs-container"} {
		c, err := LoadSudoConfigStrict(dir + "/mcp-sudo.yaml")
		if err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		for user, u := range c.Users {
			for tool, privs := range u.Privileged.Tools {
				if !ContainerTools[tool] || !privs.Allowed {
					continue
				}
				if len(privs.Containers) == 0 {
					t.Errorf("%s: user %s, tool %s: allowed with no containers list", dir, user, tool)
				}
			}
		}
	}
}
