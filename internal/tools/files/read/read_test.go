package readfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRead(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// 1. Test Line Level
	start := 2
	end := 4
	args := ReadFileArgs{
		Path:      filePath,
		StartLine: &start,
		EndLine:   &end,
	}
	b, _ := json.Marshal(args)

	res, err := Read(b)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	expectedLines := "Line 2\nLine 3\nLine 4\n"
	if res != expectedLines {
		t.Errorf("expected %q, got %q", expectedLines, res)
	}

	// 2. Test Byte Level
	offset := int64(7) // Start of "Line 2"
	limit := int64(6)  // "Line 2"
	args2 := ReadFileArgs{
		Path:   filePath,
		Offset: &offset,
		Limit:  &limit,
	}
	b2, _ := json.Marshal(args2)

	res2, err := Read(b2)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if res2 != "Line 2" {
		t.Errorf("expected 'Line 2', got %q", res2)
	}
}
