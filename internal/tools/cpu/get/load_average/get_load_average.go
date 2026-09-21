package load_average

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func GetLoadAverage(argsJSON []byte) (string, error) {
	// Let's try /proc/loadavg first for a cleaner output, fallback to uptime
	content, err := os.ReadFile("/proc/loadavg")
	if err == nil {
		return string(content), nil
	}

	cmd := exec.Command("uptime")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("uptime error: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
