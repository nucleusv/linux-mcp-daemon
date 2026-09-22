package read

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	pid := os.Getpid()

	// 1. Status
	args1 := ProcessReadArgs{PID: pid, Target: "status"}
	argsJSON1, _ := json.Marshal(args1)
	res1, err := Read(argsJSON1)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
			t.Skipf("Skipping test, procfs not available on this OS: %v", err)
		}
		t.Fatalf("Read status failed: %v", err)
	}
	if len(res1) == 0 {
		t.Errorf("Empty status result")
	}

	// 2. Cmdline
	args2 := ProcessReadArgs{PID: pid, Target: "cmdline"}
	argsJSON2, _ := json.Marshal(args2)
	res2, err := Read(argsJSON2)
	if err != nil {
		t.Fatalf("Read cmdline failed: %v", err)
	}
	if res2 == "[]" || res2 == "" {
		t.Errorf("Empty or invalid cmdline result: %s", res2)
	}

	// 3. Environ
	args3 := ProcessReadArgs{PID: pid, Target: "environ"}
	argsJSON3, _ := json.Marshal(args3)
	res3, err := Read(argsJSON3)
	if err != nil {
		t.Fatalf("Read environ failed: %v", err)
	}
	if res3 == "{}" || res3 == "" {
		t.Errorf("Empty or invalid environ result: %s", res3)
	}

	// 4. Limits
	args4 := ProcessReadArgs{PID: pid, Target: "limits"}
	argsJSON4, _ := json.Marshal(args4)
	res4, err := Read(argsJSON4)
	if err != nil {
		t.Fatalf("Read limits failed: %v", err)
	}
	if res4 == "[]" || res4 == "" || res4 == "null" {
		t.Errorf("Empty or invalid limits result: %s", res4)
	}
	if !strings.Contains(res4, "\"name\"") || !strings.Contains(res4, "\"soft\"") {
		t.Errorf("Limits result missing expected fields: %s", res4)
	}

	// 5. Open files
	args5 := ProcessReadArgs{PID: pid, Target: "open_files"}
	argsJSON5, _ := json.Marshal(args5)
	res5, err := Read(argsJSON5)
	if err != nil {
		t.Fatalf("Read open_files failed: %v", err)
	}
	if !strings.Contains(res5, "\"fd\"") || !strings.Contains(res5, "\"target\"") {
		t.Errorf("Open files result missing expected fields: %s", res5)
	}
}
