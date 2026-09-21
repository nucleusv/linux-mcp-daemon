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

type CurlArgs struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	Timeout int               `json:"timeout"`
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
	if args.Body != "" {
		reqBody = bytes.NewBufferString(args.Body)
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
