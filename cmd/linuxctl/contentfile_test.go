package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var contentSchema = map[string]interface{}{"properties": map[string]interface{}{"content": map[string]interface{}{"type": "string"}}}

func TestContentFileReadsFileAndStdinVerbatim(t *testing.T) {
	text := "# $(touch /tmp/pwned) `id` $HOME\n* * * * * echo hi; echo there | cat\n"
	p := filepath.Join(t.TempDir(), "c.txt")
	os.WriteFile(p, []byte(text), 0o600)

	f := map[string]interface{}{"content-file": p}
	if err := applyContentFlags("files/create", contentSchema, f, strings.NewReader("")); err != nil || f["content"] != text {
		t.Fatalf("file: %v %q", err, f["content"])
	}
	f = map[string]interface{}{"content-file": "-"}
	if err := applyContentFlags("cron/manage", contentSchema, f, strings.NewReader(text)); err != nil || f["content"] != text {
		t.Fatalf("stdin: %v %q", err, f["content"])
	}
	if _, left := f["content-file"]; left {
		t.Error("content-file must not reach the tool")
	}
}

func TestContentFileErrors(t *testing.T) {
	for name, c := range map[string]struct {
		tool  string
		flags map[string]interface{}
	}{
		"missing file":   {"files/create", map[string]interface{}{"content-file": "/nonexistent/x"}},
		"both given":     {"files/create", map[string]interface{}{"content-file": "-", "content": "x"}},
		"no content arg": {"cpu/list", map[string]interface{}{"content-file": "-"}},
		"bare flag":      {"files/create", map[string]interface{}{"content-file": true}},
	} {
		schema := contentSchema
		if c.tool == "cpu/list" {
			schema = map[string]interface{}{"properties": map[string]interface{}{}}
		}
		if err := applyContentFlags(c.tool, schema, c.flags, strings.NewReader("")); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestEmptyCrontabNeedsClear(t *testing.T) {
	// the accident: --content "$(cat missing.txt)" arrives as an empty string
	if err := applyContentFlags("cron/manage", contentSchema, map[string]interface{}{"content": ""}, nil); err == nil {
		t.Error("an empty crontab must be refused without --clear")
	}
	f := map[string]interface{}{"clear": true}
	if err := applyContentFlags("cron/manage", contentSchema, f, nil); err != nil || f["content"] != "" {
		t.Errorf("--clear: %v %v", err, f)
	}
	if _, left := f["clear"]; left {
		t.Error("clear must not reach the tool")
	}
	if err := applyContentFlags("cron/manage", contentSchema, map[string]interface{}{"clear": true, "content": "x"}, nil); err == nil {
		t.Error("--clear with content must be refused")
	}
	// a file write of nothing is legitimate (touch): only crontabs are guarded
	if err := applyContentFlags("files/create", contentSchema, map[string]interface{}{"content": ""}, nil); err != nil {
		t.Errorf("files/create with empty content: %v", err)
	}
	// reading a crontab sends no content at all
	if err := applyContentFlags("cron/manage", contentSchema, map[string]interface{}{}, nil); err != nil {
		t.Errorf("read: %v", err)
	}
}
