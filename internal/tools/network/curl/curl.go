package curl

import (
	"bytes"
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
	Data         string            `json:"data,omitempty"`          // Data is the request body payload.
	Insecure     bool              `json:"insecure,omitempty"`      // Insecure skips TLS certificate validation.
	OutputFormat string            `json:"output_format,omitempty"` // OutputFormat specifies the desired output format. Defaults to text.
	Timeout      int               `json:"timeout,omitempty"`       // Timeout is the request timeout in seconds. Defaults to 10.
}

type CurlResponse struct {
	StatusCode int               `json:"status_code"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
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

	client := &http.Client{
		Timeout: time.Duration(timeoutSecs) * time.Second,
	}

	var reqBody io.Reader
	if args.Data != "" {
		reqBody = bytes.NewBuffer([]byte(args.Data))
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

	bodyBytes, err := io.ReadAll(resp.Body)
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
	}

	out, err := json.MarshalIndent(outResp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode response: %v", err)
	}

	return string(out), nil
}
