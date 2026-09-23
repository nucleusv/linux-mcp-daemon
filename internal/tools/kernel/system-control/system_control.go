package system_control

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// SystemControlArgs defines the parameters for the kernel/sysctl tool.
type SystemControlArgs struct {
	Key     string `json:"key"`                // Key is the kernel parameter to read or write (e.g. "net.ipv4.ip_forward"). Required unless ReadAll is true.
	Value   string `json:"value,omitempty"`    // Value is the value to write to the parameter. If provided, writes the value.
	ReadAll bool   `json:"read_all,omitempty"` // ReadAll reads all kernel parameters (sysctl -a).
}

// validKey matches sysctl parameter names (dotted or slash-separated) and
// can never start with "-".
var validKey = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.\-/*]*$`)

func SystemControl(argsJSON []byte) (string, error) {
	var args SystemControlArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	cmdArgs := []string{}

	if args.ReadAll {
		cmdArgs = append(cmdArgs, "-a")
	} else {
		if args.Key == "" {
			return "", fmt.Errorf("key argument is required unless read_all is true")
		}
		// The key is a positional sysctl argument, so anything starting
		// with "-" would be parsed as an option - "-p/etc/shadow" makes
		// sysctl load that file and echo its lines back, as root.
		if !validKey.MatchString(args.Key) {
			return "", fmt.Errorf("invalid key %q: expected a sysctl name like net.ipv4.ip_forward", args.Key)
		}
		if strings.ContainsAny(args.Value, "\n\x00") {
			return "", fmt.Errorf("invalid value: must be a single line")
		}

		if args.Value != "" {
			// Write mode
			cmdArgs = append(cmdArgs, "-w", fmt.Sprintf("%s=%s", args.Key, args.Value))
		} else {
			// Read mode
			cmdArgs = append(cmdArgs, args.Key)
		}
	}

	cmd := exec.Command("sysctl", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("sysctl error: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
