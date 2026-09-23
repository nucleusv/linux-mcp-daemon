package ping

import (
	"context"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/netpolicy"

	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

// PingArgs defines the parameters for the network/ping tool.
type PingArgs struct {
	Host         string `json:"host"`                    // Host is the IP address or hostname to ping. Required.
	Port         int    `json:"port,omitempty"`          // Port is the TCP port to ping. Defaults to 80.
	Timeout      int    `json:"timeout,omitempty"`       // Timeout is the maximum time in seconds to wait for a reply. Defaults to 5.
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
	// NetworkPolicy is injected by the daemon from mcp-sudo.yaml (never
	// taken from the caller); nil means unrestricted.
	NetworkPolicy *netpolicy.Policy `json:"_network_policy,omitempty"`
}

type PingResponse struct {
	Host      string  `json:"host"`
	Port      int     `json:"port"`
	Success   bool    `json:"success"`
	LatencyMs float64 `json:"latency_ms"`
	Error     string  `json:"error,omitempty"`
}

func Ping(argsJSON []byte) (string, error) {
	var args PingArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	args.Host = strings.TrimSpace(args.Host)
	if args.Host == "" {
		return "", fmt.Errorf("host argument is required")
	}

	port := args.Port
	if port <= 0 {
		port = 80 // Default to HTTP port for TCP ping
	}

	timeoutSecs := args.Timeout
	if timeoutSecs <= 0 {
		timeoutSecs = 5
	}

	target := net.JoinHostPort(args.Host, fmt.Sprintf("%d", port))
	timeout := time.Duration(timeoutSecs) * time.Second

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := args.NetworkPolicy.DialContext(ctx, "tcp", target)
	latency := time.Since(start)

	resp := PingResponse{
		Host:      args.Host,
		Port:      port,
		LatencyMs: float64(latency.Microseconds()) / 1000.0,
	}

	if err != nil {
		resp.Success = false
		resp.Error = err.Error()
	} else {
		resp.Success = true
		conn.Close()
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode response: %v", err)
	}

	return string(out), nil
}
