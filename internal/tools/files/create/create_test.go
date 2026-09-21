package createfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreate(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.txt")

	// 1. Create with content
	args1 := CreateFileArgs{Path: testPath, Content: "hello"}
	argsJSON, _ := json.Marshal(args1)
	res, err := Create(argsJSON)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if !strings.Contains(res, "Successfully created") {
		t.Errorf("Unexpected result: %s", res)
	}

	content, _ := os.ReadFile(testPath)
	if string(content) != "hello" {
		t.Errorf("Content mismatch: got %q, want %q", content, "hello")
	}

	// 2. Touch without content
	args2 := CreateFileArgs{Path: testPath}
	argsJSON2, _ := json.Marshal(args2)
	res2, err := Create(argsJSON2)
	if err != nil {
		t.Fatalf("Touch failed: %v", err)
	}
	if !strings.Contains(res2, "Successfully touched") {
		t.Errorf("Unexpected result: %s", res2)
	}
}
