package curl

import (
	"github.com/nucleusv/linux-mcp-daemon/internal/netpolicy"

	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CurlArgs defines the parameters for the network/curl tool.
type CurlArgs struct {
	URL          string            `json:"url"`                     // URL is the target URL to request. Required.
	Method       string            `json:"method,omitempty"`        // Method is the HTTP method to use (e.g., "GET", "POST"). Defaults to "GET".
	Headers      map[string]string `json:"headers,omitempty"`       // Headers contains the HTTP headers to send.
	Body         string            `json:"body,omitempty"`          // Body is the request body payload (the name the tool schema advertises).
	Data         string            `json:"data,omitempty"`          // Data is an older alias for Body.
	Insecure     bool              `json:"insecure,omitempty"`      // Insecure skips TLS certificate validation.
	OutputFormat string            `json:"output_format,omitempty"` // OutputFormat specifies the desired output format. Defaults to text.
	Timeout      int               `json:"timeout,omitempty"`       // Timeout is the request timeout in seconds. Defaults to 10.
	// NetworkPolicy is injected by the daemon from mcp-sudo.yaml (never
	// taken from the caller); nil means unrestricted.
	NetworkPolicy *netpolicy.Policy `json:"_network_policy,omitempty"`
}

type CurlResponse struct {
	StatusCode int               `json:"status_code"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Truncated  bool              `json:"truncated,omitempty"` // Body was cut at the 10 MB cap.
}

func Curl(argsJSON []byte) (string, error) {
	var args CurlArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	args.URL = strings.TrimSpace(args.URL)
	if args.URL == "" {
		return "", fmt.Errorf("url argument is required")
	}

	args.Method = strings.ToUpper(strings.TrimSpace(args.Method))
	if args.Method == "" {
		args.Method = "GET"
	}

	timeoutSecs := args.Timeout
	if timeoutSecs <= 0 {
		timeoutSecs = 10
	}

	// Every connection - including each redirect hop - goes through the
	// policy dialer, which checks the resolved IP and dials exactly it.
	transport := &http.Transport{
		DialContext:         args.NetworkPolicy.DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	if args.NetworkPolicy == nil {
		// Unrestricted: keep the default proxy-from-environment behavior.
		// With a policy, no proxy - it would make the policy check the
		// proxy's address instead of the real destination.
		transport.Proxy = http.ProxyFromEnvironment
	}
	if args.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client := &http.Client{
		Timeout:   time.Duration(timeoutSecs) * time.Second,
		Transport: transport,
	}

	// The schema advertises "body"; this used to read only "data", so every
	// POST body sent through the tool was silently dropped.
	payload := args.Body
	if payload == "" {
		payload = args.Data
	}
	var reqBody io.Reader
	if payload != "" {
		reqBody = strings.NewReader(payload)
	}

	req, err := http.NewRequest(args.Method, args.URL, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	for k, v := range args.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %v", err)
	}
	defer resp.Body.Close()

	// Cap the body: an unbounded ReadAll of a huge or endless response
	// would exhaust the worker's memory.
	const maxBody = 10 << 20
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	truncated := len(bodyBytes) > maxBody
	if truncated {
		bodyBytes = bodyBytes[:maxBody]
	}
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		respHeaders[k] = strings.Join(v, ", ")
	}

	outResp := CurlResponse{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    respHeaders,
		Body:       string(bodyBytes),
		Truncated:  truncated,
	}

	// No HTML escaping: bodies are usually HTML, and Go's default turns
	// every < > & into \u003c-style escapes.
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(outResp); err != nil {
		return "", fmt.Errorf("failed to encode response: %v", err)
	}
	return strings.TrimSuffix(out.String(), "\n"), nil
}
