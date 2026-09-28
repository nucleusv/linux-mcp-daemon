package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestCompleteMcpdAndReload(t *testing.T) {
	granted := Registry{Tools: []ToolDef{{Name: "daemon/reload-config", ToolsGroup: "daemon", LinuxctlVerb: "reload"}}}
	cases := []struct {
		reg   Registry
		words []string
		want  []string
	}{
		{Registry{}, []string{"edit", ""}, []string{"mcpd"}},
		{Registry{}, []string{"edit", "mcpd", ""}, []string{"config"}},
		{Registry{}, []string{"edit", "mcpd", "config", ""}, []string{"daemon", "sudo", "users"}},
		{Registry{}, []string{"create", "mcpd", ""}, []string{"user"}},
		{granted, []string{"reload", ""}, []string{"daemon"}},
	}
	for _, c := range cases {
		if got := completeWords(c.reg, c.words); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: got %v, want %v", c.words, got, c.want)
		}
	}
	if contains(completeWords(Registry{}, []string{""}), "reload") {
		t.Error("reload offered without the tool in the registry (not granted)")
	}
	if !contains(completeWords(granted, []string{""}), "reload") {
		t.Error("reload not offered with the tool granted")
	}
}

func TestResolveUngrantedGroup(t *testing.T) {
	_, err := Resolve(Registry{}, "reload", "daemon", nil)
	if err == nil || !strings.Contains(err.Error(), "daemon/reload-config granted") {
		t.Errorf("reload daemon without the tool: err = %v", err)
	}
	reg := Registry{Tools: []ToolDef{{Name: "files/list", ToolsGroup: "files", LinuxctlVerb: "list"}}}
	if _, err := Resolve(reg, "frobnicate", "files", nil); err == nil || !strings.Contains(err.Error(), `no verb "frobnicate" in group "files"`) {
		t.Errorf("unknown verb in a known group: err = %v", err)
	}
}

// A group's bare-reachable tool is also nameable by its own command name, so
// `get docker containers` must resolve exactly like `get docker` - not swallow
// "containers" as an unmatched positional and warn about it (FR-015).
func TestResolveGetAcceptsDefaultToolCommandName(t *testing.T) {
	reg := Registry{Tools: []ToolDef{
		{Name: "docker/containers", ToolsGroup: "docker", LinuxctlVerb: "get",
			InputSchema: map[string]interface{}{"properties": map[string]interface{}{"pattern": map[string]interface{}{"type": "string"}}}},
		{Name: "docker/images", ToolsGroup: "docker", LinuxctlVerb: "images"},
		{Name: "files/read", ToolsGroup: "files", LinuxctlVerb: "get",
			InputSchema: map[string]interface{}{
				"properties": map[string]interface{}{"path": map[string]interface{}{"type": "string"}},
				"required":   []interface{}{"path"},
			}},
		{Name: "files/list", ToolsGroup: "files", LinuxctlVerb: "list"},
	}}
	cases := []struct {
		verb, group string
		rest        []string
		wantTool    string
		wantPos     []string
	}{
		{"get", "docker", []string{"containers"}, "docker/containers", nil},
		{"get", "docker", nil, "docker/containers", nil},
		{"get", "docker", []string{"images"}, "docker/images", nil},
		{"get", "files", []string{"read", "/etc/hosts"}, "files/read", []string{"/etc/hosts"}},
		{"get", "files", []string{"/etc/hosts"}, "files/read", []string{"/etc/hosts"}},
		{"get", "files", []string{"list", "/var/log"}, "files/list", []string{"/var/log"}},
	}
	for _, c := range cases {
		action, err := Resolve(reg, c.verb, c.group, c.rest)
		if err != nil {
			t.Errorf("%s %s %v: %v", c.verb, c.group, c.rest, err)
			continue
		}
		if action.Tool.Name != c.wantTool {
			t.Errorf("%s %s %v: tool = %q, want %q", c.verb, c.group, c.rest, action.Tool.Name, c.wantTool)
		}
		if !reflect.DeepEqual(action.Positional, c.wantPos) && !(len(action.Positional) == 0 && len(c.wantPos) == 0) {
			t.Errorf("%s %s %v: positional = %v, want %v", c.verb, c.group, c.rest, action.Positional, c.wantPos)
		}
	}
}

// dockerFixture is a minimal but realistic docker group: a bare-reachable
// read tool with no positional slot of its own (docker/containers, per the
// real schema in internal/rpc/docker.go), one keyword-only tool, and four
// resource templates - matching the group FR-015 was found and fixed
// against.
func dockerFixture() Registry {
	return Registry{
		Tools: []ToolDef{
			{Name: "docker/containers", ToolsGroup: "docker", LinuxctlVerb: "get",
				InputSchema: map[string]interface{}{"properties": map[string]interface{}{"pattern": map[string]interface{}{"type": "string"}}}},
			{Name: "docker/images", ToolsGroup: "docker", LinuxctlVerb: "images"},
		},
		Templates: []TemplateDef{
			{URITemplate: "container://{name}/{view}", Name: "Docker Container Introspection", Group: "docker", LinuxctlVerb: "container"},
			{URITemplate: "image://{name}/inspect", Name: "Docker Image Inspect", Group: "docker", LinuxctlVerb: "image"},
			{URITemplate: "volume://{name}/inspect", Name: "Docker Volume Inspect", Group: "docker", LinuxctlVerb: "volume"},
			{URITemplate: "docker-network://{name}/inspect", Name: "Docker Network Inspect", Group: "docker", LinuxctlVerb: "network"},
		},
		Resources: []ResourceDef{
			{URI: "network://interfaces", Name: "Network Interfaces", Group: "network", LinuxctlVerb: "interfaces"},
		},
	}
}

// FR-015: `get <group> <keyword> <name>` must resolve to a matching resource
// template, not silently fall through to the group's bare-reachable tool
// with the keyword and name dropped as ignored extra positionals.
func TestResolveGetMatchesTemplateByKeyword(t *testing.T) {
	reg := dockerFixture()
	cases := []struct {
		rest     []string
		wantURI  string
		wantPos  []string
	}{
		{[]string{"network", "appnet"}, "docker-network://{name}/inspect", []string{"appnet"}},
		{[]string{"volume", "app-data"}, "volume://{name}/inspect", []string{"app-data"}},
		{[]string{"image", "nginx"}, "image://{name}/inspect", []string{"nginx"}},
		{[]string{"container", "web-1", "status"}, "container://{name}/{view}", []string{"web-1", "status"}},
	}
	for _, c := range cases {
		action, err := Resolve(reg, "get", "docker", c.rest)
		if err != nil {
			t.Errorf("get docker %v: %v", c.rest, err)
			continue
		}
		if action.Kind != "template_read" {
			t.Errorf("get docker %v: kind = %q, want template_read (tool %q)", c.rest, action.Kind, action.Tool.Name)
			continue
		}
		if action.Template.URITemplate != c.wantURI {
			t.Errorf("get docker %v: template = %q, want %q", c.rest, action.Template.URITemplate, c.wantURI)
		}
		if !reflect.DeepEqual(action.Positional, c.wantPos) {
			t.Errorf("get docker %v: positional = %v, want %v", c.rest, action.Positional, c.wantPos)
		}
	}
}

// FR-015: a keyword that matches no tool, template or resource in the group
// is a hard error, never a silent match against the bare-reachable tool with
// the keyword surviving as an ignored "extra argument".
func TestResolveGetUnmatchedKeywordIsError(t *testing.T) {
	reg := dockerFixture()
	_, err := Resolve(reg, "get", "docker", []string{"networkz"}) // typo for "network"/"networks"
	if err == nil {
		t.Fatal("get docker networkz: expected an error, got a resolved action")
	}
	if !strings.Contains(err.Error(), `no read target "networkz" in group "docker"`) {
		t.Errorf("get docker networkz: err = %v", err)
	}
}

// FR-015: network:// (the host's own networking) must keep resolving by
// `get network interfaces` and never collide with docker-network://, even
// though a Docker network can be named "interfaces" - the two live in
// different groups (network vs docker) and matchTemplateByKeyword only
// looks at templates in the group being resolved.
func TestResolveGetHostNetworkNotAmbiguousWithDockerNetwork(t *testing.T) {
	reg := dockerFixture()
	action, err := Resolve(reg, "get", "network", []string{"interfaces"})
	if err != nil {
		t.Fatalf("get network interfaces: %v", err)
	}
	if action.Kind != "resource_read" || action.ResourceURI != "network://interfaces" {
		t.Errorf("get network interfaces: kind=%q uri=%q, want resource_read network://interfaces", action.Kind, action.ResourceURI)
	}
}
