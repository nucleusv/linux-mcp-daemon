package connections

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// ssStates maps accepted spellings to ss(8)'s state names. Callers
// naturally use the netstat/kernel spelling ("LISTEN", "TIME_WAIT"), which
// ss rejects ("wrong state name") - it only knows its own lowercase,
// hyphenated names.
var ssStates = map[string]string{
	"listen":       "listening",
	"listening":    "listening",
	"established":  "established",
	"syn-sent":     "syn-sent",
	"syn-recv":     "syn-recv",
	"syn-received": "syn-recv",
	"fin-wait-1":   "fin-wait-1",
	"fin-wait1":    "fin-wait-1",
	"fin-wait-2":   "fin-wait-2",
	"fin-wait2":    "fin-wait-2",
	"time-wait":    "time-wait",
	"closed":       "closed",
	"close":        "closed",
	"close-wait":   "close-wait",
	"last-ack":     "last-ack",
	"closing":      "closing",
	// ss's own state groups
	"all":          "all",
	"connected":    "connected",
	"synchronized": "synchronized",
}

func ssState(state string) (string, error) {
	key := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(state)), "_", "-")
	if s, ok := ssStates[key]; ok {
		return s, nil
	}
	var valid []string
	for k := range ssStates {
		valid = append(valid, k)
	}
	sort.Strings(valid)
	return "", fmt.Errorf("unknown connection state %q (valid, case-insensitive: %s)", state, strings.Join(valid, ", "))
}

// GetConnectionsArgs defines the parameters for the network/connections tool.
type GetConnectionsArgs struct {
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
	State        string `json:"state,omitempty"`         // State filters by connection state (e.g., "ESTABLISHED", "LISTEN").
	Port         int    `json:"port,omitempty"`          // Port filters by a specific local or remote port.
	Privileged   bool   `json:"privileged,omitempty"`    // Privileged runs the tool as root to see all processes owning the sockets.
}

func Connections(argsJSON []byte) (string, error) {
	var args GetConnectionsArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	// -a, not -l: the tool reports active connections as well as listening
	// ports. A state filter narrows it down (e.g. "listening").
	cmdArgs := []string{"-tuanp"} // TCP, UDP, all states, numeric, show processes
	if args.State != "" {
		state, err := ssState(args.State)
		if err != nil {
			return "", err
		}
		cmdArgs = append(cmdArgs, "state", state)
	}
	if args.Port > 0 {
		cmdArgs = append(cmdArgs, fmt.Sprintf("( sport = :%d or dport = :%d )", args.Port, args.Port))
	}

	cmd := exec.Command("ss", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ss error: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
