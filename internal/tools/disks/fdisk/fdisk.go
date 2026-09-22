package fdisk

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type FdiskArgs struct {
	Device string `json:"device,omitempty"`
}

func Fdisk(args []byte) (string, error) {
	var params FdiskArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	// -l lists partition tables.
	cmdArgs := []string{"-l"}
	if params.Device != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("/dev/%s", params.Device))
	}

	cmd := exec.Command("fdisk", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("fdisk failed: %v, output: %s", err, string(output))
	}

	// fdisk outputs text, so we return it as a raw string.
	return string(output), nil
}
