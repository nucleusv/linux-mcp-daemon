package ping

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

type PingArgs struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`    // Optional, defaults to 80 or 443
	Timeout int    `json:"timeout"` // Timeout in seconds
}

type PingResponse struct {
	Host       string  `json:"host"`
	Port       int     `json:"port"`
	Success    bool    `json:"success"`
	LatencyMs  float64 `json:"latency_ms"`
	Error      string  `json:"error,omitempty"`
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

	target := fmt.Sprintf("%s:%d", args.Host, port)
	timeout := time.Duration(timeoutSecs) * time.Second

	start := time.Now()
	conn, err := net.DialTimeout("tcp", target, timeout)
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
