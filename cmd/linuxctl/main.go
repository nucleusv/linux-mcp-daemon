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
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"gopkg.in/yaml.v3"
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

	// 2. Extract dynamic group/command and arguments
	args := flag.Args()
	var groupCommand string
	if len(args) > 0 {
		groupCommand = args[0]
	}
	
	var group, command string
	if strings.Contains(groupCommand, "/") {
		parts := strings.SplitN(groupCommand, "/", 2)
		group = parts[0]
		command = parts[1]
	} else {
		group = groupCommand
		if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
			command = args[1]
		}
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
					if group == "stdio" {
						// Transparently forward all JSON-RPC messages to stdout for Claude Desktop
						fmt.Println(data)
					} else {
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
	if group == "stdio" {
		// Read JSON-RPC requests from stdin line by line
		scanner := bufio.NewScanner(os.Stdin)
		// Claude Desktop sends one JSON object per line
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
			
			// Forward the exact raw JSON to the message endpoint
			postReq, err := http.NewRequest("POST", postEndpoint, strings.NewReader(line))
			if err != nil {
				continue
			}
			postReq.Header.Set("Authorization", "Bearer "+authToken)
			postReq.Header.Set("Content-Type", "application/json")
			postResp, err := client.Do(postReq)
			if err == nil {
				postResp.Body.Close()
			}
		}
		os.Exit(0)
	}

	if command == "" {
		if group == "ping" {
			fmt.Println("Successfully connected to mcpd daemon!")
			os.Exit(0)
		}
		
		if group == "resources" {
			// Fetch resources/list
			resourcesResp := callMethod(authToken, "1", "resources/list", nil)
			var resResult map[string]interface{}
			json.Unmarshal(resourcesResp.Result, &resResult)
			
			resList, ok := resResult["resources"].([]interface{})
			if !ok {
				fmt.Println("Failed to fetch resources from daemon.")
				os.Exit(1)
			}
			fmt.Println("Available static resources:")
			for _, r := range resList {
				res := r.(map[string]interface{})
				fmt.Printf("  %-20s - %s\n", res["uri"], res["description"])
			}
			
			resTemplatesResp := callMethod(authToken, "2", "resources/templates/list", nil)
			var templatesResult map[string]interface{}
			json.Unmarshal(resTemplatesResp.Result, &templatesResult)

			resTemplates, ok := templatesResult["resourceTemplates"].([]interface{})
			if ok && len(resTemplates) > 0 {
				fmt.Println("\nAvailable resource templates:")
				for _, r := range resTemplates {
					res := r.(map[string]interface{})
					fmt.Printf("  %-20s - %s\n", res["uriTemplate"], res["description"])
				}
			}
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

		if group == "" {
			fmt.Println("Usage: linuxctl [options] <group>/<command> [command-options]")
			fmt.Println("       linuxctl [options] resources")
			fmt.Println("       linuxctl [options] resource <uri>")
			fmt.Println("Options:")
			flag.PrintDefaults()
			
			fmt.Println("\nAvailable tool groups (dynamically fetched from daemon):")
			
			groups := make(map[string]bool)
			for _, t := range toolList {
				tool := t.(map[string]interface{})
				if tg, ok := tool["tools_group"].(string); ok {
					groups[tg] = true
				}
			}
			
			for g := range groups {
				fmt.Printf("  %s\n", g)
			}
			fmt.Println("\nRun 'linuxctl <group>' to see available commands in that group.")
			fmt.Println("Run 'linuxctl resources' to see available resources.")
			os.Exit(1)
		} else if group != "resource" {
			// Print commands matching the group
			fmt.Printf("Available commands for group '%s':\n", group)
			found := false
			for _, t := range toolList {
				tool := t.(map[string]interface{})
				if tg, ok := tool["tools_group"].(string); ok && tg == group {
					found = true
					name := tool["name"].(string)
					desc := tool["description"].(string)
					
					displayName := strings.TrimPrefix(name, group+"/")
					
					fmt.Printf("  %-20s - %s\n", displayName, desc)
				}
			}
			if !found {
				fmt.Printf("  (No commands found for group '%s')\n", group)
			}
			os.Exit(1)
		}
	}

	// 6. Dynamic tools/call
	expectedCommand := group + "/" + command
	
	var respRPC JSONRPCResponse
	var outputFormat string
	
	if group == "resource" {
		if len(args) < 2 {
			fmt.Println("Error: resource URI required. Example: linuxctl resource os://uname")
			os.Exit(1)
		}
		uri := args[1]
		
		// Parse formatting arguments
		for i := 2; i < len(args); i++ {
			if args[i] == "--output" || args[i] == "-o" {
				if i+1 < len(args) {
					outputFormat = args[i+1]
				}
			}
		}
		
		params := map[string]interface{}{
			"uri": uri,
		}
		respRPC = callMethod(authToken, "2", "resources/read", params)
	} else {
		// Fetch tools/list to find the exact match
		tools := callMethod(authToken, "1", "tools/list", nil)
		var resultList map[string]interface{}
		json.Unmarshal(tools.Result, &resultList)
		
		var actualToolName string
		var actualTool map[string]interface{}
		if toolList, ok := resultList["tools"].([]interface{}); ok {
			for _, t := range toolList {
				tool := t.(map[string]interface{})
				_ = tool["tools_group"].(string)
				name := tool["name"].(string)
				
				if name == expectedCommand {
					actualToolName = name
					actualTool = tool
					break
				}
			}
		}
		
		if actualToolName == "" {
			fmt.Printf("Error: Command '%s/%s' not found.\n", group, command)
			os.Exit(1)
		}
	
		// Build arguments from CLI args like --path /var/log --privileged true
		toolArgs := make(map[string]interface{})
		var positionalArgs []string
		
		startIndex := 1
		if len(args) > 1 && args[1] == command {
			startIndex = 2
		}
		
		for i := startIndex; i < len(args); i++ {
			arg := args[i]
			if strings.HasPrefix(arg, "--") {
				key := strings.TrimPrefix(arg, "--")
				key = strings.TrimPrefix(key, "-")
				if key == "o" || key == "output" {
					key = "output_format"
					if i+1 < len(args) {
						outputFormat = args[i+1]
					}
				}
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
					val := args[i+1]
					if val == "true" {
						toolArgs[key] = true
					} else if val == "false" {
						toolArgs[key] = false
					} else if num, err := strconv.Atoi(val); err == nil {
						toolArgs[key] = num
					} else {
						toolArgs[key] = val
					}
					i++ // skip value
				} else {
					toolArgs[key] = true // boolean flag
				}
			} else {
				positionalArgs = append(positionalArgs, arg)
			}
		}
	
		// Smart mapping of positional arguments
		if len(positionalArgs) > 0 {
			if schema, ok := actualTool["inputSchema"].(map[string]interface{}); ok {
				if props, ok := schema["properties"].(map[string]interface{}); ok {
					// If the tool has a "path" property, map the first positional arg to it
					if _, hasPath := props["path"]; hasPath && toolArgs["path"] == nil {
						toolArgs["path"] = positionalArgs[0]
					}
				}
			}
		}
	
		params := map[string]interface{}{
			"name":      actualToolName,
			"arguments": toolArgs,
		}
	
		respRPC = callMethod(authToken, "2", "tools/call", params)
	}

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
				if outputFormat == "json" {
					// Prettify JSON if it is valid JSON
					var obj interface{}
					if err := json.Unmarshal([]byte(text), &obj); err == nil {
						b, _ := json.MarshalIndent(obj, "", "  ")
						fmt.Println(string(b))
					} else {
						fmt.Print(text)
					}
				} else if outputFormat == "yaml" {
					var obj interface{}
					if err := json.Unmarshal([]byte(text), &obj); err == nil {
						b, _ := yaml.Marshal(obj)
						fmt.Print(string(b))
					} else {
						fmt.Print(text)
					}
				} else if outputFormat == "table" || outputFormat == "wide" {
					// Basic dynamic table formatting for JSON arrays/objects
					var obj interface{}
					if err := json.Unmarshal([]byte(text), &obj); err == nil {
						w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
						
						if arr, ok := obj.([]interface{}); ok && len(arr) > 0 {
							// Array of objects
							if first, ok := arr[0].(map[string]interface{}); ok {
								var keys []string
								for k := range first {
									keys = append(keys, k)
								}
								fmt.Fprintln(w, strings.ToUpper(strings.Join(keys, "\t")))
								for _, item := range arr {
									if m, ok := item.(map[string]interface{}); ok {
										var vals []string
										for _, k := range keys {
											vals = append(vals, fmt.Sprintf("%v", m[k]))
										}
										fmt.Fprintln(w, strings.Join(vals, "\t"))
									}
								}
							} else {
								// Array of primitives
								for _, item := range arr {
									fmt.Fprintln(w, fmt.Sprintf("%v", item))
								}
							}
						} else if m, ok := obj.(map[string]interface{}); ok {
							// Single object
							for k, v := range m {
								fmt.Fprintf(w, "%s\t%v\n", strings.ToUpper(k), v)
							}
						}
						w.Flush()
					} else {
						fmt.Print(text)
					}
				} else {
					fmt.Print(text)
				}
			}
		}
	} else if contentList, ok := result["contents"].([]interface{}); ok {
		// Handle resource/read result array which is called "contents" not "content"
		for _, c := range contentList {
			content := c.(map[string]interface{})
			if text, ok := content["text"].(string); ok {
				if outputFormat == "json" {
					var obj interface{}
					if err := json.Unmarshal([]byte(text), &obj); err == nil {
						b, _ := json.MarshalIndent(obj, "", "  ")
						fmt.Println(string(b))
					} else {
						fmt.Print(text)
					}
				} else if outputFormat == "yaml" {
					var obj interface{}
					if err := json.Unmarshal([]byte(text), &obj); err == nil {
						b, _ := yaml.Marshal(obj)
						fmt.Print(string(b))
					} else {
						fmt.Print(text)
					}
				} else if outputFormat == "table" || outputFormat == "wide" {
					var obj interface{}
					if err := json.Unmarshal([]byte(text), &obj); err == nil {
						w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
						if arr, ok := obj.([]interface{}); ok && len(arr) > 0 {
							if first, ok := arr[0].(map[string]interface{}); ok {
								var keys []string
								for k := range first {
									keys = append(keys, k)
								}
								fmt.Fprintln(w, strings.ToUpper(strings.Join(keys, "\t")))
								for _, item := range arr {
									if m, ok := item.(map[string]interface{}); ok {
										var vals []string
										for _, k := range keys {
											vals = append(vals, fmt.Sprintf("%v", m[k]))
										}
										fmt.Fprintln(w, strings.Join(vals, "\t"))
									}
								}
							} else {
								for _, item := range arr {
									fmt.Fprintln(w, fmt.Sprintf("%v", item))
								}
							}
						} else if m, ok := obj.(map[string]interface{}); ok {
							for k, v := range m {
								fmt.Fprintf(w, "%s\t%v\n", strings.ToUpper(k), v)
							}
						}
						w.Flush()
					} else {
						fmt.Print(text)
					}
				} else {
					fmt.Print(text)
				}
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
