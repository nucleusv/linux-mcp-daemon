package tracepath

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
)

type TracePathArgs struct {
	Host    string `json:"host"`
	MaxHops int    `json:"max_hops,omitempty"`
}

var validHost = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.:_-]*$`)

func TracePath(args []byte) (string, error) {
	var params TracePathArgs
	if len(args) == 0 {
		return "", fmt.Errorf("missing arguments: host is required")
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if params.Host == "" {
		return "", fmt.Errorf("host parameter is required")
	}

	// Positional argument to traceroute - reject anything that could be
	// parsed as an option.
	if !validHost.MatchString(params.Host) {
		return "", fmt.Errorf("invalid host %q: expected a hostname or IP address", params.Host)
	}
	if params.MaxHops > 255 {
		params.MaxHops = 255
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
