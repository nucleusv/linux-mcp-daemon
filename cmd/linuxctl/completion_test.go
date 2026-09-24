package main

import (
	"reflect"
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
