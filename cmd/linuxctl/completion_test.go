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
