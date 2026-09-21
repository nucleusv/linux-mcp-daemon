package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

var (
	serverURL = flag.String("server", "http://localhost:9090", "The URL of the mcpd server")
	token     = flag.String("token", "", "Bearer token for authentication")
)

type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var (
	postEndpoint  string
	responseChans = make(map[string]chan JSONRPCResponse)
	chanMutex     sync.Mutex
)

func main() {
	flag.Parse()

	// 1. Get auth token
	authToken := *token
	if authToken == "" {
		authToken = os.Getenv("MCP_TOKEN")
		if authToken == "" {
			fmt.Println("Error: Authentication token is required.")
			fmt.Println("Provide it via -token flag or MCP_TOKEN environment variable.")
			os.Exit(1)
		}
	}

	// 2. Extract dynamic verb, resource, and arguments
	args := flag.Args()
	var verb, resource string
	if len(args) > 0 {
		verb = args[0]
	}
	if len(args) > 1 {
		resource = args[1]
	}

	// 3. Connect to SSE
	req, err := http.NewRequest("GET", *serverURL+"/sse", nil)
	if err != nil {
		fmt.Printf("Failed to create SSE request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to connect to daemon: %v\n", err)
		os.Exit(1)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Daemon returned status: %s\n%s\n", resp.Status, string(body))
		os.Exit(1)
	}

	// 4. Start SSE reader loop
	go func() {
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		var eventName string

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					fmt.Printf("\nSSE stream error: %v\n", err)
				}
				return
			}
			line = strings.TrimSpace(line)

			if strings.HasPrefix(line, "event: ") {
				eventName = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if eventName == "endpoint" {
					postEndpoint = *serverURL + data
				} else if eventName == "message" {
					var rpcResp JSONRPCResponse
					if err := json.Unmarshal([]byte(data), &rpcResp); err == nil {
						chanMutex.Lock()
						if ch, ok := responseChans[rpcResp.ID]; ok {
							ch <- rpcResp
							delete(responseChans, rpcResp.ID)
						}
						chanMutex.Unlock()
					}
				}
			} else if line == "" {
				eventName = ""
			}
		}
	}()

	// Wait for endpoint
	for postEndpoint == "" {
		// active busy wait for simplicity, should be quick
	}

	// 5. Route Logic
	if resource == "" {
		if verb == "ping" {
			fmt.Println("Successfully connected to mcpd daemon!")
			os.Exit(0)
		}
		
		// Fetch tools/list for dynamic help
		tools := callMethod(authToken, "1", "tools/list", nil)
		var result map[string]interface{}
		json.Unmarshal(tools.Result, &result)
		
		toolList, ok := result["tools"].([]interface{})
		if !ok {
			fmt.Println("Failed to fetch tools from daemon.")
			os.Exit(1)
		}

		// Helper to extract verb and resource from tool name
		parseToolName := func(toolName string) (string, string) {
			parts := strings.SplitN(toolName, "_", 2)
			if len(parts) == 2 {
				return parts[0], strings.ReplaceAll(parts[1], "_", "-")
			}
			return "run", toolName // fallback verb if no underscore
		}

		if verb == "" {
			fmt.Println("Usage: linuxctl [options] <verb> <resource> [command-options]")
			fmt.Println("Options:")
			flag.PrintDefaults()
			
			fmt.Println("\nAvailable verbs (dynamically fetched from daemon):")
			
			verbs := make(map[string]bool)
			for _, t := range toolList {
				tool := t.(map[string]interface{})
				name := tool["name"].(string)
				v, _ := parseToolName(name)
				verbs[v] = true
			}
			
			for v := range verbs {
				fmt.Printf("  %s\n", v)
			}
			fmt.Println("\nRun 'linuxctl <verb>' to see available resources for that action.")
			os.Exit(1)
		} else {
			// Print resources matching the verb
			fmt.Printf("Available resources for action '%s':\n", verb)
			found := false
			for _, t := range toolList {
				tool := t.(map[string]interface{})
				name := tool["name"].(string)
				v, res := parseToolName(name)
				if v == verb {
					found = true
					desc := tool["description"].(string)
					fmt.Printf("  %-20s - %s\n", res, desc)
				}
			}
			if !found {
				fmt.Printf("  (No resources found for action '%s')\n", verb)
			}
			os.Exit(1)
		}
	}

	// 6. Dynamic tools/call
	// Reconstruct the exact backend tool name
	toolName := verb + "_" + strings.ReplaceAll(resource, "-", "_")

	// Build arguments from CLI args like --path /var/log --privileged true
	toolArgs := make(map[string]interface{})
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				val := args[i+1]
				if val == "true" {
					toolArgs[key] = true
				} else if val == "false" {
					toolArgs[key] = false
				} else {
					toolArgs[key] = val
				}
				i++ // skip value
			} else {
				toolArgs[key] = true // boolean flag
			}
		}
	}

	params := map[string]interface{}{
		"name":      toolName,
		"arguments": toolArgs,
	}

	respRPC := callMethod(authToken, "2", "tools/call", params)
	
	if respRPC.Error != nil {
		fmt.Printf("Error: %s (Code: %d)\n", respRPC.Error.Message, respRPC.Error.Code)
		os.Exit(1)
	}

	var result map[string]interface{}
	json.Unmarshal(respRPC.Result, &result)
	
	if contentList, ok := result["content"].([]interface{}); ok {
		for _, c := range contentList {
			content := c.(map[string]interface{})
			if text, ok := content["text"].(string); ok {
				fmt.Print(text)
			}
		}
	} else if result["isError"] == true {
		fmt.Println("Tool executed with an error, but no specific text was provided.")
		os.Exit(1)
	} else {
		// Dump raw result if we can't parse it beautifully
		b, _ := json.MarshalIndent(respRPC.Result, "", "  ")
		fmt.Println(string(b))
	}
}

func callMethod(authToken string, id string, method string, params interface{}) JSONRPCResponse {
	reqRPC := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	body, _ := json.Marshal(reqRPC)

	req, _ := http.NewRequest("POST", postEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")

	ch := make(chan JSONRPCResponse, 1)
	chanMutex.Lock()
	responseChans[id] = ch
	chanMutex.Unlock()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to POST JSON-RPC: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		fmt.Printf("Failed to submit request (HTTP %d): %s\n", resp.StatusCode, string(b))
		os.Exit(1)
	}

	// Block until response arrives from SSE loop
	return <-ch
}
