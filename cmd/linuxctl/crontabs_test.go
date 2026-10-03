package main

import (
	"reflect"
	"testing"
)

func cronFixture() Registry {
	return Registry{
		Tools: []ToolDef{
			{Name: "cron/manage", ToolsGroup: "crontabs", LinuxctlVerb: "get", InputSchema: map[string]interface{}{
				"properties": map[string]interface{}{
					"user": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"},
					"if_match": map[string]interface{}{"type": "string"}, "privileged": map[string]interface{}{"type": "boolean"},
				}, "required": []interface{}{}}},
			// a tool with both pid and user: pid must still win the positional word
			{Name: "processes/list", ToolsGroup: "processes", LinuxctlVerb: "get", InputSchema: map[string]interface{}{
				"properties": map[string]interface{}{"pid": map[string]interface{}{"type": "integer"}, "user": map[string]interface{}{"type": "string"}}}},
		},
		Templates: []TemplateDef{{URITemplate: "crontab://{user}/{view}", Name: "User Crontab", Group: "crontabs", LinuxctlVerb: "crontab"}},
	}
}

func TestCrontabsGetReadsOwnOrNamedUser(t *testing.T) {
	reg := cronFixture()
	for _, c := range []struct {
		words []string
		user  interface{}
	}{{nil, nil}, {[]string{"test_user"}, "test_user"}} {
		a, err := Resolve(reg, "get", "crontabs", c.words)
		if err != nil || a.Kind != "tool_call" || a.Tool.Name != "cron/manage" {
			t.Fatalf("get crontabs %v: %+v %v", c.words, a, err)
		}
		args := map[string]interface{}{}
		if rest := mapPositionalArgs(a.Tool.InputSchema, args, a.Positional); len(rest) != 0 || args["user"] != c.user {
			t.Errorf("get crontabs %v: args %v rest %v", c.words, args, rest)
		}
	}
}

func TestCrontabsUpdateIsTheWrite(t *testing.T) {
	reg := cronFixture()
	for _, words := range [][]string{nil, {"test_user"}} {
		a, err := Resolve(reg, "update", "crontabs", words)
		if err != nil || a.Tool.Name != "cron/manage" {
			t.Fatalf("update crontabs %v: %+v %v", words, a, err)
		}
		// with --content already set from a flag, the positional word is the user
		args := map[string]interface{}{"content": "0 3 * * * x"}
		rest := mapPositionalArgs(a.Tool.InputSchema, args, a.Positional)
		if len(rest) != 0 || (len(words) > 0 && args["user"] != "test_user") || args["content"] != "0 3 * * * x" {
			t.Errorf("update crontabs %v: %v %v", words, args, rest)
		}
	}
}

func TestCrontabsDescribeAggregatesInfoAndText(t *testing.T) {
	reg := cronFixture()
	a, err := Resolve(reg, "describe", "crontabs", []string{"test_user"})
	if err != nil || a.Kind != "template_read" {
		t.Fatalf("describe crontabs: %+v %v", a, err)
	}
	uris, extra := buildDescribeURIs(a.Template, a.Positional)
	if !reflect.DeepEqual(uris, []string{"crontab://test_user/info", "crontab://test_user/text"}) || len(extra) != 0 {
		t.Errorf("uris %v extra %v", uris, extra)
	}
	_, extra = buildDescribeURIs(a.Template, []string{"test_user", "oops"})
	if !reflect.DeepEqual(extra, []string{"oops"}) {
		t.Errorf("an extra word must be reported, got %v", extra)
	}
}

func TestPositionalUserDoesNotStealOtherTools(t *testing.T) {
	reg := cronFixture()
	a, err := Resolve(reg, "get", "processes", []string{"1234"})
	if err != nil {
		t.Fatal(err)
	}
	args := map[string]interface{}{}
	mapPositionalArgs(a.Tool.InputSchema, args, a.Positional)
	if args["pid"] != "1234" && args["pid"] != 1234 || args["user"] != nil {
		t.Errorf("get processes 1234 must fill pid, got %v", args)
	}
}

// 0.5.0 regression: adding "user" to the positional priority list made
// `exec docker web-1 -- echo hello` put web-1 into docker/exec's optional
// `user` and treat "echo" as the container.
func TestExecDockerKeepsContainerAheadOfUser(t *testing.T) {
	schema := map[string]interface{}{
		"properties": map[string]interface{}{
			"container": map[string]interface{}{"type": "string"},
			"command":   map[string]interface{}{"type": "array"},
			"user":      map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"container", "command"},
	}
	args := map[string]interface{}{}
	rest := mapPositionalArgs(schema, args, []string{"web-1", "echo", "hello"})
	if len(rest) != 0 || args["container"] != "web-1" || !reflect.DeepEqual(args["command"], []interface{}{"echo", "hello"}) {
		t.Errorf("args %v rest %v", args, rest)
	}
	if _, set := args["user"]; set {
		t.Errorf("user must stay unset: %v", args)
	}
	// an explicit --user is kept and the positionals still fill the required ones
	args = map[string]interface{}{"user": "root"}
	mapPositionalArgs(schema, args, []string{"web-1", "id"})
	if args["container"] != "web-1" || args["user"] != "root" {
		t.Errorf("explicit user: %v", args)
	}
}
