package traceroute

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

type TracerouteArgs struct {
	Host    string `json:"host"`
	MaxHops int    `json:"max_hops,omitempty"`
}

func Traceroute(args []byte) (string, error) {
	var params TracerouteArgs
	if len(args) == 0 {
		return "", fmt.Errorf("missing arguments: host is required")
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if params.Host == "" {
		return "", fmt.Errorf("host parameter is required")
	}

	cmdArgs := []string{}
	if params.MaxHops > 0 {
		cmdArgs = append(cmdArgs, "-m", strconv.Itoa(params.MaxHops))
	}
	cmdArgs = append(cmdArgs, params.Host)

	cmd := exec.Command("traceroute", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("traceroute failed: %v, output: %s", err, string(output))
	}

	return string(output), nil
}
