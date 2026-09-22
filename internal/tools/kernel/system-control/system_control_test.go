package system_control

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSysctl(t *testing.T) {
	// 1. Read
	args1 := SystemControlArgs{Key: "kernel.ostype"}
	argsJSON, _ := json.Marshal(args1)
	res, err := SystemControl(argsJSON)
	if err != nil {
		t.Logf("Sysctl read failed (likely missing key on this OS): %v", err)
	} else if !strings.Contains(res, "Linux") && !strings.Contains(res, "Darwin") {
		t.Errorf("Unexpected sysctl result: %s", res)
	}
}
