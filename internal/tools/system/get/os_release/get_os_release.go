package os_release

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func GetOSRelease(argsJSON []byte) (string, error) {
	var out bytes.Buffer

	// Try reading /etc/os-release
	osRelease, err := os.ReadFile("/etc/os-release")
	if err == nil {
		out.WriteString("OS Release Info:\n")
		out.Write(osRelease)
		out.WriteString("\n")
	}

	// Also get kernel info
	cmd := exec.Command("uname", "-a")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	if err == nil {
		out.WriteString("Kernel Info:\n")
		out.Write(stdout.Bytes())
	} else {
		out.WriteString(fmt.Sprintf("Failed to run uname: %v\n", err))
	}

	if out.Len() == 0 {
		return "", fmt.Errorf("failed to retrieve any OS release information")
	}

	return out.String(), nil
}
