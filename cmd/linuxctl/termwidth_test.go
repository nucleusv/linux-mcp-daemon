package main

import (
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestObjectKeys(t *testing.T) {
	got := objectKeys([]byte(`{"pid":1,"user":"root","nested":{"z":1,"a":[1,"x"]},"cmdline":"init"}`))
	if strings.Join(got, ",") != "pid,user,nested,cmdline" {
		t.Errorf("objectKeys = %v", got)
	}
	if objectKeys([]byte(`[1,2]`)) != nil {
		t.Error("objectKeys on an array must be nil")
	}
}

// captureStdout runs f and returns what it printed.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	b, _ := io.ReadAll(r)
	return string(b)
}

func TestTableColumnsFollowEachArraysOwnOrder(t *testing.T) {
	// Regression: "ni" and "cpu_percent" appear in the summary before the
	// processes array, and a document-wide key order put them first.
	doc := `{"summary":{"cpu_percent":{"us":1,"ni":0},"users":1},"processes":[{"pid":7,"user":"root","ni":0,"cpu_percent":2.5,"cmdline":"x"},{"pid":8,"user":"u","ni":5,"cpu_percent":0}]}`
	out := captureStdout(t, func() { printFormatted(doc, "wide") })
	lines := strings.Split(out, "\n")
	if !strings.HasPrefix(lines[0], "SUMMARY") || !strings.Contains(lines[0], `{"cpu_percent":{"us":1,"ni":0},"users":1}`) {
		t.Errorf("summary line keeps server order: %q", lines[0])
	}
	header := strings.Fields(lines[3])
	if strings.Join(header, " ") != "PID USER NI CPU_PERCENT CMDLINE" {
		t.Errorf("columns = %v", header)
	}
	if strings.Contains(out, "<nil>") {
		t.Errorf("missing field rendered as <nil>:\n%s", out)
	}
}

func TestTruncateLine(t *testing.T) {
	cases := []struct {
		in    string
		width int
		want  string
	}{
		{"short\n", 10, "short\n"},
		{"exactly10!\n", 10, "exactly10!\n"},
		{"this is too long\n", 10, "this is t…\n"},
		{"no limit at all\n", 0, "no limit at all\n"},
		{"├─ünïcödé-long\n", 5, "├─ün…\n"},
	}
	for _, c := range cases {
		if got := truncateLine(c.in, c.width); got != c.want {
			t.Errorf("truncateLine(%q, %d) = %q, want %q", c.in, c.width, got, c.want)
		}
	}
}

func TestPositionalFillsRequiredFields(t *testing.T) {
	schema := map[string]interface{}{
		"properties": map[string]interface{}{
			"path": map[string]interface{}{"type": "string"},
			"mode": map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"path", "mode"},
	}
	args := map[string]interface{}{}
	rest := mapPositionalArgs(schema, args, []string{"/srv/app", "0755"})
	if args["path"] != "/srv/app" || args["mode"] != "0755" || len(rest) != 0 {
		t.Errorf("args %v, leftover %v", args, rest)
	}
}

// A unix timestamp is a valid value for docker/logs `since`, declared string.
// The flag parser reads it as a number before the schema is known, so the
// coercion has to put it back - and must leave a genuine integer field alone.
func TestCoerceFlagTypes(t *testing.T) {
	schema := map[string]interface{}{
		"properties": map[string]interface{}{
			"since": map[string]interface{}{"type": "string"},
			"lines": map[string]interface{}{"type": "integer"},
		},
	}
	args := map[string]interface{}{"since": 1759005000, "lines": 50, "unknown": 7}
	coerceFlagTypes(schema, args)
	if args["since"] != "1759005000" {
		t.Errorf("since = %#v, want the string", args["since"])
	}
	if args["lines"] != 50 || args["unknown"] != 7 {
		t.Errorf("non-string fields should be untouched, got %v", args)
	}
}

// An array parameter (docker/exec command) has no flag syntax of its own, so
// it arrives as text: JSON, or a bare token meaning a one-element argv.
func TestCoerceFlagTypesArray(t *testing.T) {
	schema := map[string]interface{}{
		"properties": map[string]interface{}{
			"command": map[string]interface{}{"type": "array"},
		},
	}
	args := map[string]interface{}{"command": `["sh","-c","ls /app"]`}
	coerceFlagTypes(schema, args)
	if want := []interface{}{"sh", "-c", "ls /app"}; !reflect.DeepEqual(args["command"], want) {
		t.Errorf("command = %#v, want %#v", args["command"], want)
	}

	bare := map[string]interface{}{"command": "hostname"}
	coerceFlagTypes(schema, bare)
	if want := []interface{}{"hostname"}; !reflect.DeepEqual(bare["command"], want) {
		t.Errorf("bare command = %#v, want %#v", bare["command"], want)
	}
}

// The same array parameter, filled positionally. Bare words are the argv;
// a lone JSON array is that array - which used to be passed on as one literal
// word, so `exec docker fr011-probe '["hostname"]'` ran a program actually
// named `["hostname"]` and exited 127.
func TestPositionalArrayTakesJSONOrWords(t *testing.T) {
	schema := map[string]interface{}{
		"properties": map[string]interface{}{
			"container": map[string]interface{}{"type": "string"},
			"command":   map[string]interface{}{"type": "array"},
		},
		"required": []interface{}{"container", "command"},
	}
	for _, tc := range []struct {
		positional []string
		want       []interface{}
	}{
		{[]string{"web-1", "sh", "-c", "ls /app"}, []interface{}{"sh", "-c", "ls /app"}},
		{[]string{"web-1", `["sh","-c","ls /app"]`}, []interface{}{"sh", "-c", "ls /app"}},
		{[]string{"web-1", "hostname"}, []interface{}{"hostname"}},
	} {
		args := map[string]interface{}{}
		mapPositionalArgs(schema, args, tc.positional)
		if args["container"] != "web-1" || !reflect.DeepEqual(args["command"], tc.want) {
			t.Errorf("%v -> %#v, want command %#v", tc.positional, args, tc.want)
		}
	}
}
