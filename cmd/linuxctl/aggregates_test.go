package main

import (
	"reflect"
	"testing"
)

func TestBuildDescribeURIsLeftover(t *testing.T) {
	cases := []struct {
		name  string
		tpl   string
		args  []string
		uris  int
		extra []string
	}{
		{"simple extra", "docker://networks/{name}", []string{"bridge", "x"}, 1, []string{"x"}},
		{"simple exact", "docker://networks/{name}", []string{"bridge"}, 1, nil},
		{"file extra, no double warn", "file:///{path}", []string{"/etc/hosts", "x"}, 2, []string{"x"}},
		{"file exact", "file:///{path}", []string{"/etc/hosts"}, 2, nil},
		{"process pid only", "process://{pid}/{target}", []string{"1"}, 3, nil},
		{"process extra", "process://{pid}/{target}", []string{"1", "x"}, 3, []string{"x"}},
		{"container view", "container://{name}/{view}", []string{"web", "logs"}, 1, nil},
		{"container extra", "container://{name}/{view}", []string{"web", "logs", "x"}, 1, []string{"x"}},
		{"service extra", "service://{name}/status", []string{"sshd", "x"}, 1, []string{"x"}},
	}
	for _, c := range cases {
		uris, extra := buildDescribeURIs(TemplateDef{URITemplate: c.tpl}, c.args)
		if len(uris) != c.uris || (len(extra) != 0 || len(c.extra) != 0) && !reflect.DeepEqual(extra, c.extra) {
			t.Errorf("%s: got %d uris, extra %v; want %d, %v", c.name, len(uris), extra, c.uris, c.extra)
		}
	}
}
