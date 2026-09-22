package smartctl

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type SmartctlArgs struct {
	Device string `json:"device"`
}

func Smartctl(args []byte) (string, error) {
	var params SmartctlArgs
	if len(args) == 0 {
		return "", fmt.Errorf("missing arguments: device is required")
	}
	
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if params.Device == "" {
		return "", fmt.Errorf("device parameter is required")
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
