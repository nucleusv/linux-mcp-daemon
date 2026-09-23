package updatefile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func run(t *testing.T, args map[string]interface{}) {
	t.Helper()
	b, _ := json.Marshal(args)
	if _, err := Update(b); err != nil {
		t.Fatalf("Update(%s): %v", b, err)
	}
}

func TestReplaceKeepsTrailingNewlineThenAppend(t *testing.T) {
	// The exact sequence that produced "line threeline four" on the VPS.
	path := filepath.Join(t.TempDir(), "f.txt")
	os.WriteFile(path, []byte("line one\nline two\nline three\n"), 0o600)

	run(t, map[string]interface{}{"path": path, "start_line": 2, "end_line": 2, "content": "line TWO (updated)"})
	run(t, map[string]interface{}{"path": path, "append": true, "content": "line four (appended)\n"})

	got, _ := os.ReadFile(path)
	want := "line one\nline TWO (updated)\nline three\nline four (appended)\n"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Errorf("permissions changed to %v, want 0600", info.Mode().Perm())
	}
}

func TestReplaceWithoutTrailingNewlineStaysWithout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.txt")
	os.WriteFile(path, []byte("a\nb\nc"), 0o644)
	run(t, map[string]interface{}{"path": path, "start_line": 3, "end_line": 3, "content": "C\n"})
	if got, _ := os.ReadFile(path); string(got) != "a\nb\nC" {
		t.Errorf("got %q, want %q", got, "a\nb\nC")
	}
}

func TestReplaceMultiLineAndBeyondEnd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.txt")
	os.WriteFile(path, []byte("1\n2\n3\n4\n"), 0o644)
	run(t, map[string]interface{}{"path": path, "start_line": 2, "end_line": 3, "content": "x\ny\nz"})
	run(t, map[string]interface{}{"path": path, "start_line": 99, "end_line": 99, "content": "end"})
	if got, _ := os.ReadFile(path); string(got) != "1\nx\ny\nz\n4\nend\n" {
		t.Errorf("got %q", got)
	}
}

func TestLongLine(t *testing.T) {
	// bufio.Scanner used to fail on lines over 64KB.
	path := filepath.Join(t.TempDir(), "f.txt")
	long := make([]byte, 100000)
	for i := range long {
		long[i] = 'x'
	}
	os.WriteFile(path, append(append([]byte("first\n"), long...), '\n'), 0o644)
	run(t, map[string]interface{}{"path": path, "start_line": 1, "end_line": 1, "content": "FIRST"})
	got, _ := os.ReadFile(path)
	if string(got) != "FIRST\n"+string(long)+"\n" {
		t.Errorf("long line mangled (len %d)", len(got))
	}
}

// Original test, kept from before the trailing-newline fix. Its expected
// value used to end in "line4" with no newline - encoding the bug itself -
// and now ends in "line4\n", matching the input file's own newline style.
func TestUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testPath, []byte("line1\nline2\nline3\n"), 0644)

	// 1. Append
	args1 := UpdateFileArgs{Path: testPath, Content: "line4\n", Append: true}
	argsJSON1, _ := json.Marshal(args1)
	if _, err := Update(argsJSON1); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// 2. Replace line2 (start=2, end=2)
	start, end := 2, 2
	args2 := UpdateFileArgs{Path: testPath, Content: "line2_replaced", StartLine: &start, EndLine: &end}
	argsJSON2, _ := json.Marshal(args2)
	if _, err := Update(argsJSON2); err != nil {
		t.Fatalf("Replace failed: %v", err)
	}

	content, _ := os.ReadFile(testPath)
	expected := "line1\nline2_replaced\nline3\nline4\n"
	if string(content) != expected {
		t.Errorf("Content mismatch:\nGot: %q\nWant: %q", string(content), expected)
	}
}
