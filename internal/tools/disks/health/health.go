package health

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
)

type HealthArgs struct {
	Device string `json:"device"`
}

var validDevice = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

func Health(args []byte) (string, error) {
	var params HealthArgs
	if len(args) == 0 {
		return "", fmt.Errorf("missing arguments: device is required")
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if params.Device == "" {
		return "", fmt.Errorf("device parameter is required")
	}

	// A bare device name (sda, nvme0n1, sg1) - never a path, so
	// "../etc/shadow" can't point smartctl (as root) outside /dev.
	if !validDevice.MatchString(params.Device) {
		return "", fmt.Errorf("invalid device %q: expected a block device name like sda or nvme0n1", params.Device)
	}
	cmdArgs := []string{"-j", "-a", fmt.Sprintf("/dev/%s", params.Device)}
	cmd := exec.Command("smartctl", cmdArgs...)

	// smartctl often returns non-zero exit codes even for success (e.g., bitmask flags),
	// so we shouldn't fail purely on err != nil if we got valid JSON.
	output, _ := cmd.CombinedOutput()

	// Try to validate if output is valid JSON
	var js map[string]interface{}
	if err := json.Unmarshal(output, &js); err != nil {
		return "", fmt.Errorf("smartctl failed or returned invalid json: %s", string(output))
	}

	return string(output), nil
}
