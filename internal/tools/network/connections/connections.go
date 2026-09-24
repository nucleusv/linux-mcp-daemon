package connections

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/nucleusv/linux-mcp-daemon/internal/kernel"
)

// stateFilters maps accepted spellings - ss's names and the netstat/kernel
// ones ("LISTEN", "TIME_WAIT") - to the set of socket states they select.
// "listening" includes unconnected UDP sockets: that's what's listening on
// a UDP port.
var stateFilters = map[string][]string{
	"listen":       {"LISTEN", "UNCONN"},
	"listening":    {"LISTEN", "UNCONN"},
	"established":  {"ESTAB"},
	"estab":        {"ESTAB"},
	"syn-sent":     {"SYN-SENT"},
	"syn-recv":     {"SYN-RECV"},
	"syn-received": {"SYN-RECV"},
	"fin-wait-1":   {"FIN-WAIT-1"},
	"fin-wait1":    {"FIN-WAIT-1"},
	"fin-wait-2":   {"FIN-WAIT-2"},
	"fin-wait2":    {"FIN-WAIT-2"},
	"time-wait":    {"TIME-WAIT"},
	"closed":       {"UNCONN"},
	"close":        {"UNCONN"},
	"unconn":       {"UNCONN"},
	"close-wait":   {"CLOSE-WAIT"},
	"last-ack":     {"LAST-ACK"},
	"closing":      {"CLOSING"},
	// ss's state groups
	"all":          nil,
	"connected":    {"ESTAB", "SYN-SENT", "SYN-RECV", "FIN-WAIT-1", "FIN-WAIT-2", "TIME-WAIT", "CLOSE-WAIT", "LAST-ACK", "CLOSING"},
	"synchronized": {"ESTAB", "SYN-RECV", "FIN-WAIT-1", "FIN-WAIT-2", "TIME-WAIT", "CLOSE-WAIT", "LAST-ACK", "CLOSING"},
}

// stateSet returns the states a filter selects; nil means all.
func stateSet(state string) (map[string]bool, error) {
	key := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(state)), "_", "-")
	states, ok := stateFilters[key]
	if !ok {
		var valid []string
		for k := range stateFilters {
			valid = append(valid, k)
		}
		sort.Strings(valid)
		return nil, fmt.Errorf("unknown connection state %q (valid, case-insensitive: %s)", state, strings.Join(valid, ", "))
	}
	if states == nil {
		return nil, nil
	}
	set := map[string]bool{}
	for _, s := range states {
		set[s] = true
	}
	return set, nil
}

// GetConnectionsArgs defines the parameters for the network/connections tool.
type GetConnectionsArgs struct {
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
	State        string `json:"state,omitempty"`         // State filters by connection state (e.g., "ESTABLISHED", "LISTEN").
	Port         int    `json:"port,omitempty"`          // Port filters by a specific local or remote port.
	Privileged   bool   `json:"privileged,omitempty"`    // Privileged runs the tool as root to see all processes owning the sockets.
}

// Connections lists TCP and UDP sockets in every state, like `ss -tuanp`,
// read from /proc/net and /proc/<pid>/fd rather than by running ss.
func Connections(argsJSON []byte) (string, error) {
	var args GetConnectionsArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}
	var states map[string]bool
	if args.State != "" {
		var err error
		if states, err = stateSet(args.State); err != nil {
			return "", err
		}
	}

	socks, err := kernel.ReadSockets("/proc")
	if err != nil {
		return "", err
	}
	owners := kernel.SocketOwners("/proc")

	var out []kernel.Socket
	for _, s := range socks {
		if states != nil && !states[s.State] {
			continue
		}
		if args.Port > 0 && s.LocalPort != args.Port && s.PeerPort != args.Port {
			continue
		}
		if s.Inode != 0 {
			s.Processes = owners[s.Inode]
		}
		out = append(out, s)
	}

	switch args.OutputFormat {
	case "json", "yaml", "table", "wide":
		if out == nil {
			out = []kernel.Socket{}
		}
		b, err := json.Marshal(out)
		return string(b), err
	}
	return format(out), nil
}

// format prints sockets in ss -tuanp's layout.
func format(socks []kernel.Socket) string {
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Netid\tState\tRecv-Q\tSend-Q\tLocal Address:Port\tPeer Address:Port\tProcess")
	for _, s := range socks {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\t%s\n",
			s.Netid, s.State, s.RecvQ, s.SendQ, s.Endpoint(true), s.Endpoint(false), users(s.Processes))
	}
	w.Flush()
	return b.String()
}

// users formats owners as ss does: users:(("sshd",pid=812,fd=3)).
func users(procs []kernel.Process) string {
	if len(procs) == 0 {
		return ""
	}
	parts := make([]string, len(procs))
	for i, p := range procs {
		parts[i] = fmt.Sprintf("(%q,pid=%d,fd=%d)", p.Name, p.PID, p.FD)
	}
	return "users:(" + strings.Join(parts, ",") + ")"
}
