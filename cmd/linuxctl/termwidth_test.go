package main

import (
	"io"
	"os"
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
