package updatefile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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
	expected := "line1\nline2_replaced\nline3\nline4"
	if string(content) != expected {
		t.Errorf("Content mismatch:\nGot: %q\nWant: %q", string(content), expected)
	}
}
