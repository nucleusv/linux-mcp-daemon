package health

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
)

type HealthArgs struct {
	Device     string `json:"device"`
	Privileged bool   `json:"privileged,omitempty"` // Reading SMART data needs root.
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

	// smartctl often returns non-zero exit codes even for success (its exit
	// status is a bitmask of drive conditions), so a non-zero exit isn't an
	// error by itself as long as it produced JSON. Failing to run at all is,
	// and used to surface only as a confusing empty "invalid json".
	output, runErr := cmd.CombinedOutput()
	if runErr != nil {
		var exitErr *exec.ExitError
		if !errors.As(runErr, &exitErr) {
			if errors.Is(runErr, exec.ErrNotFound) {
				return "", fmt.Errorf("smartctl is not installed on this host - install the smartmontools package (in containerized deployments, privileged calls run in the host's filesystem, so it must be installed on the host)")
			}
			return "", fmt.Errorf("failed to run smartctl: %v", runErr)
		}
	}

	var js map[string]interface{}
	if err := json.Unmarshal(output, &js); err != nil {
		return "", fmt.Errorf("smartctl failed or returned invalid json: %s", string(output))
	}

	return string(output), nil
}
