package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

var (
	serverURL  = flag.String("server", "http://localhost:9091", "The URL of the mcpd server")
	token      = flag.String("token", "", "Bearer token for authentication")
	configPath = flag.String("config-path", "./configs", "Local path to the daemon's configs/ directory (used only by the local-only 'mcpd' admin group)")
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
	idCounter     int
)

func nextID() string {
	idCounter++
	return strconv.Itoa(idCounter)
}

func main() {
	flag.Parse()
	rawArgs := flag.Args()

	// The "mcpd" group is local-only admin (user/token management) - it never
	// talks to the daemon over the network, so it's handled before any auth
	// token or server connection is set up, and works with no token at all.
	if len(rawArgs) >= 2 && rawArgs[1] == "mcpd" {
		handleMcpdAdmin(append([]string{rawArgs[0]}, rawArgs[2:]...))
		return
	}

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

	firstWord := ""
	if len(rawArgs) > 0 {
		firstWord = rawArgs[0]
	}

	// 2. Connect to SSE
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

	// 3. Start SSE reader loop
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
					if firstWord == "stdio" {
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

	// 4. Transport-bridge and meta commands, unrelated to the verb/group grammar.
	if firstWord == "stdio" {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
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

	if firstWord == "ping" {
		fmt.Println("Successfully connected to mcpd daemon!")
		os.Exit(0)
	}

	// "get mcp <tools|resources|prompts>" - a reserved meta-group for the MCP
	// protocol's own top-level catalog concepts, not a real tools_group.
	// Distinct from "tool <name>"/"resource <uri>" (direct single-item
	// invocation by identifier) - "mcp" is specifically for discovering
	// what's available. mcpd currently implements tools and resources
	// (incl. resource templates); prompts is a real MCP capability this
	// daemon doesn't implement, reported clearly rather than silently
	// omitted, since "what can we get from mcpd over the MCP standard" is
	// exactly the question this command answers.
	if firstWord == "get" && len(rawArgs) > 1 && rawArgs[1] == "mcp" {
		if len(rawArgs) < 3 {
			fmt.Println("Usage: linuxctl get mcp <tools|resources|prompts|info>")
			os.Exit(1)
		}
		switch rawArgs[2] {
		case "tools":
			printToolsList(fetchRegistry(authToken))
		case "resources":
			printResourcesList(authToken)
		case "prompts":
			printPromptsList(authToken)
		case "info":
			printMCPInfo(authToken)
		default:
			fmt.Printf("Error: unknown mcp catalog %q (expected tools, resources, prompts, or info)\n", rawArgs[2])
			os.Exit(1)
		}
		os.Exit(0)
	}

	if firstWord == "resource" {
		if len(rawArgs) < 2 {
			fmt.Println("Error: resource URI required. Example: linuxctl resource os://uname")
			os.Exit(1)
		}
		_, positional, outputFormat := splitFlagsAndPositional(rawArgs[1:])
		if len(positional) == 0 {
			fmt.Println("Error: resource URI required. Example: linuxctl resource os://uname")
			os.Exit(1)
		}
		respRPC := callMethod(authToken, nextID(), "resources/read", map[string]interface{}{"uri": positional[0]})
		renderResponse(respRPC, outputFormat, "contents")
		os.Exit(0)
	}

	// "tool <name> [--flag val ...]" - direct-by-literal-name escape hatch,
	// symmetric with "resource <uri>": both call the underlying MCP method
	// (tools/call vs resources/read) directly by its exact identifier,
	// bypassing the verb/group grammar entirely. Also doubles as the
	// explicit backward-compat path to the pre-redesign flat tool-name
	// passthrough for anything the resolver doesn't (yet) handle - see
	// plan/linuxctl-redesign.md's "Open decisions" #1.
	if firstWord == "tool" {
		runToolByName(authToken, rawArgs[1:])
		os.Exit(0)
	}

	reg := fetchRegistry(authToken)

	if firstWord == "explain" {
		runExplain(reg, rawArgs[1:])
		os.Exit(0)
	}

	if len(rawArgs) < 2 {
		printTopLevelUsage(reg)
		os.Exit(1)
	}

	verb, group := rawArgs[0], rawArgs[1]
	flagArgs, positional, outputFormat := splitFlagsAndPositional(rawArgs[2:])

	action, err := Resolve(reg, verb, group, positional)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	switch action.Kind {
	case "tool_call":
		toolArgs := map[string]interface{}{}
		for k, v := range action.Args {
			toolArgs[k] = v
		}
		remaining := mapPositionalArgs(action.Tool.InputSchema, toolArgs, action.Positional)
		for k, v := range flagArgs {
			toolArgs[k] = v
		}
		if len(remaining) > 0 {
			fmt.Printf("Warning: %d extra argument(s) ignored: %s\n", len(remaining), strings.Join(remaining, " "))
		}
		respRPC := callMethod(authToken, nextID(), "tools/call", map[string]interface{}{
			"name":      action.Tool.Name,
			"arguments": toolArgs,
		})
		renderResponse(respRPC, outputFormat, "content")

	case "resource_read":
		respRPC := callMethod(authToken, nextID(), "resources/read", map[string]interface{}{"uri": action.ResourceURI})
		renderResponse(respRPC, outputFormat, "contents")

	case "template_read":
		runDescribe(authToken, action.Template, action.Positional, outputFormat)

	case "top_snapshot":
		runTop(authToken, outputFormat)
	}
}

// printToolsList lists every tool by its literal <group>/<command> name,
// grouped by tools_group - symmetric with printResourcesList, and the
// direct-by-name counterpart to "tool <name>" the same way "resources" is
// the counterpart to "resource <uri>".
func printToolsList(reg Registry) {
	byGroup := map[string][]ToolDef{}
	var groupOrder []string
	for _, t := range reg.Tools {
		if _, seen := byGroup[t.ToolsGroup]; !seen {
			groupOrder = append(groupOrder, t.ToolsGroup)
		}
		byGroup[t.ToolsGroup] = append(byGroup[t.ToolsGroup], t)
	}
	fmt.Println("Available tools:")
	for _, g := range groupOrder {
		for _, t := range byGroup[g] {
			fmt.Printf("  %-25s - %s\n", t.Name, t.Description)
		}
	}
}

// printPromptsList calls the real MCP prompts/list method rather than
// silently skipping it - mcpd doesn't implement the prompts capability
// (see internal/rpc/rpc.go's method dispatch and its initialize response,
// which only declares tools and resources), so this reports that plainly
// instead of pretending the catalog is just empty.
func printPromptsList(authToken string) {
	respRPC := callMethod(authToken, nextID(), "prompts/list", nil)
	if respRPC.Error != nil {
		fmt.Printf("This mcpd daemon does not implement the MCP prompts capability (%s).\n", respRPC.Error.Message)
		fmt.Println("Its initialize response only declares \"tools\" and \"resources\" - see `linuxctl get mcp info`.")
		return
	}
	var result map[string]interface{}
	json.Unmarshal(respRPC.Result, &result)
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}

// printMCPInfo calls the real MCP initialize method and prints the raw
// response - the server's own self-declared protocol version and
// capabilities, straight from the handshake rather than a client-side
// summary of it.
func printMCPInfo(authToken string) {
	respRPC := callMethod(authToken, nextID(), "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]interface{}{"name": "linuxctl", "version": "1.0.0"},
	})
	if respRPC.Error != nil {
		fmt.Printf("Error: %s (Code: %d)\n", respRPC.Error.Message, respRPC.Error.Code)
		os.Exit(1)
	}
	var result map[string]interface{}
	json.Unmarshal(respRPC.Result, &result)
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}

func printResourcesList(authToken string) {
	resourcesResp := callMethod(authToken, nextID(), "resources/list", nil)
	var resResult map[string]interface{}
	json.Unmarshal(resourcesResp.Result, &resResult)

	resList, ok := resResult["resources"].([]interface{})
	if !ok {
		fmt.Println("Failed to fetch resources from daemon.")
		return
	}
	fmt.Println("Available static resources:")
	for _, r := range resList {
		res := r.(map[string]interface{})
		fmt.Printf("  %-20s - %s\n", res["uri"], res["description"])
	}

	resTemplatesResp := callMethod(authToken, nextID(), "resources/templates/list", nil)
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
}

func fetchRegistry(authToken string) Registry {
	toolsResp := callMethod(authToken, nextID(), "tools/list", nil)
	var toolsResult map[string]interface{}
	json.Unmarshal(toolsResp.Result, &toolsResult)
	toolsList, _ := toolsResult["tools"].([]interface{})

	resResp := callMethod(authToken, nextID(), "resources/list", nil)
	var resResult map[string]interface{}
	json.Unmarshal(resResp.Result, &resResult)
	resList, _ := resResult["resources"].([]interface{})

	tplResp := callMethod(authToken, nextID(), "resources/templates/list", nil)
	var tplResult map[string]interface{}
	json.Unmarshal(tplResp.Result, &tplResult)
	tplList, _ := tplResult["resourceTemplates"].([]interface{})

	return BuildRegistry(toolsList, resList, tplList)
}

func printTopLevelUsage(reg Registry) {
	fmt.Println("Usage: linuxctl [options] <verb> <group> [target-keyword] [args]")
	fmt.Println("       linuxctl [options] describe <group> [target-keyword] <name>")
	fmt.Println("       linuxctl [options] explain <group> [verb]")
	fmt.Println("       linuxctl [options] get tools")
	fmt.Println("       linuxctl [options] tool <group>/<command> [--flag val ...]")
	fmt.Println("       linuxctl [options] resources")
	fmt.Println("       linuxctl [options] resource <uri>")
	fmt.Println("Options:")
	flag.PrintDefaults()

	groups := map[string]bool{}
	for _, t := range reg.Tools {
		groups[t.ToolsGroup] = true
	}
	for _, r := range reg.Resources {
		groups[r.Group] = true
	}
	fmt.Println("\nAvailable groups (dynamically fetched from daemon):")
	for g := range groups {
		fmt.Printf("  %s\n", g)
	}
	fmt.Println("\nRun 'linuxctl explain <group>' to see available verbs/keywords in that group.")
}

// runExplain is the meta-verb from plan/linuxctl-redesign.md's "True
// exceptions - no group at all" - it describes the schema itself, never
// nested under a data domain, mirroring `kubectl explain` exactly.
func runExplain(reg Registry, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: linuxctl explain <group>")
		return
	}
	group := args[0]
	found := false
	for _, t := range reg.Tools {
		if t.ToolsGroup != group {
			continue
		}
		found = true
		switch {
		case isMutationOnly(t):
			fmt.Printf("  <action>%-13s-> tool %-25s %s\n", "", t.Name, t.Description)
		case t.LinuxctlVerb == "create" || t.LinuxctlVerb == "update" || t.LinuxctlVerb == "delete":
			fmt.Printf("  %-24s -> tool %-25s %s\n", t.LinuxctlVerb, t.Name, t.Description)
		case t.LinuxctlVerb == "get":
			fmt.Printf("  get %-20s -> tool %-25s %s (bare - no keyword needed)\n", "", t.Name, t.Description)
		case t.LinuxctlVerb != "":
			fmt.Printf("  get %-20s -> tool %-25s %s\n", t.LinuxctlVerb, t.Name, t.Description)
		default:
			fmt.Printf("  %-24s -> tool %-25s %s\n", t.LinuxctlVerb, t.Name, t.Description)
		}
	}
	for _, r := range reg.Resources {
		if r.Group == group {
			found = true
			fmt.Printf("  get %-20s -> resource %-25s %s\n", r.LinuxctlVerb, r.URI, r.Description)
		}
	}
	for _, tpl := range reg.Templates {
		if tpl.Group == group {
			found = true
			kw := tpl.LinuxctlVerb
			if kw == "" {
				kw = "<name>"
			}
			fmt.Printf("  describe %-16s -> template %-25s %s\n", kw, tpl.URITemplate, tpl.Description)
		}
	}
	if !found {
		fmt.Printf("No tools, resources, or templates found in group %q.\n", group)
	}
}

// runToolByName calls a tool directly by its literal <group>/<command> name,
// symmetric with how "resource <uri>" reads a resource directly by URI -
// bypasses the verb/group resolver entirely.
func runToolByName(authToken string, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: linuxctl tool <group>/<command> [--flag val ...]")
		os.Exit(1)
	}
	toolName := args[0]
	flagArgs, positional, outputFormat := splitFlagsAndPositional(args[1:])

	reg := fetchRegistry(authToken)
	var tool ToolDef
	found := false
	for _, t := range reg.Tools {
		if t.Name == toolName {
			tool = t
			found = true
			break
		}
	}
	if !found {
		fmt.Printf("Error: tool %q not found.\n", toolName)
		os.Exit(1)
	}
	mapPositionalArgs(tool.InputSchema, flagArgs, positional)
	respRPC := callMethod(authToken, nextID(), "tools/call", map[string]interface{}{
		"name":      tool.Name,
		"arguments": flagArgs,
	})
	renderResponse(respRPC, outputFormat, "content")
}

// splitFlagsAndPositional separates --flag value pairs (and the --output/-o
// formatting flag) from plain positional tokens - keywords and target names
// are never flag-prefixed, so this must run before Resolve sees `rest`.
func splitFlagsAndPositional(args []string) (map[string]interface{}, []string, string) {
	flagArgs := map[string]interface{}{}
	var positional []string
	outputFormat := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			if key == "output" {
				if i+1 < len(args) {
					outputFormat = args[i+1]
					// Also forwarded as the tool's own output_format
					// argument: many tools (system/packages, memory/usage,
					// disks/free, ...) return fundamentally different data
					// in json mode versus a human-text summary, not just a
					// different rendering of the same data - --output isn't
					// purely a client-side display choice.
					flagArgs["output_format"] = outputFormat
					i++
				}
				continue
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				val := args[i+1]
				if val == "true" {
					flagArgs[key] = true
				} else if val == "false" {
					flagArgs[key] = false
				} else if num, err := strconv.Atoi(val); err == nil {
					flagArgs[key] = num
				} else {
					flagArgs[key] = val
				}
				i++
			} else {
				flagArgs[key] = true
			}
		} else if arg == "-o" {
			if i+1 < len(args) {
				outputFormat = args[i+1]
				flagArgs["output_format"] = outputFormat
				i++
			}
		} else {
			positional = append(positional, arg)
		}
	}
	return flagArgs, positional, outputFormat
}

// formatCell renders a JSON-decoded value for table/wide output. Go's default
// %v on a float64 (every JSON number decodes as float64) switches to
// scientific notation past a certain magnitude (e.g. disk byte counts render
// as "6.2671097856e+10" instead of "62671097856") - this keeps whole numbers
// in plain decimal instead.
func formatCell(v interface{}) string {
	if f, ok := v.(float64); ok && f == math.Trunc(f) {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return fmt.Sprintf("%v", v)
}

// renderResponse prints a JSON-RPC result's text content, formatted per
// outputFormat. contentKey is "content" for tools/call, "contents" for
// resources/read - the two methods use different field names for otherwise
// identically-shaped results.
func renderResponse(respRPC JSONRPCResponse, outputFormat, contentKey string) {
	if respRPC.Error != nil {
		fmt.Printf("Error: %s (Code: %d)\n", respRPC.Error.Message, respRPC.Error.Code)
		os.Exit(1)
	}

	var result map[string]interface{}
	json.Unmarshal(respRPC.Result, &result)

	contentList, ok := result[contentKey].([]interface{})
	if !ok {
		if result["isError"] == true {
			fmt.Println("Tool executed with an error, but no specific text was provided.")
			os.Exit(1)
		}
		b, _ := json.MarshalIndent(respRPC.Result, "", "  ")
		fmt.Println(string(b))
		return
	}

	for _, c := range contentList {
		content, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		text, ok := content["text"].(string)
		if !ok {
			continue
		}
		printFormatted(text, outputFormat)
	}
}

func printFormatted(text, outputFormat string) {
	switch outputFormat {
	case "json":
		var obj interface{}
		if err := json.Unmarshal([]byte(text), &obj); err == nil {
			b, _ := json.MarshalIndent(obj, "", "  ")
			fmt.Println(string(b))
		} else {
			fmt.Print(text)
		}
	case "yaml":
		var obj interface{}
		if err := json.Unmarshal([]byte(text), &obj); err == nil {
			b, _ := yaml.Marshal(obj)
			fmt.Print(string(b))
		} else {
			fmt.Print(text)
		}
	case "table", "wide":
		var obj interface{}
		if err := json.Unmarshal([]byte(text), &obj); err != nil {
			fmt.Print(text)
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

		printArrayAsTable := func(arr []interface{}) {
			if len(arr) == 0 {
				return
			}
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
							vals = append(vals, formatCell(m[k]))
						}
						fmt.Fprintln(w, strings.Join(vals, "\t"))
					}
				}
			} else {
				for _, item := range arr {
					fmt.Fprintln(w, formatCell(item))
				}
			}
		}

		if arr, ok := obj.([]interface{}); ok && len(arr) > 0 {
			printArrayAsTable(arr)
		} else if m, ok := obj.(map[string]interface{}); ok {
			var nestedArrays []struct {
				key string
				arr []interface{}
			}
			for k, v := range m {
				if v != nil {
					if arr, ok := v.([]interface{}); ok && len(arr) > 0 {
						if _, isObj := arr[0].(map[string]interface{}); isObj {
							nestedArrays = append(nestedArrays, struct {
								key string
								arr []interface{}
							}{k, arr})
							continue
						}
					}
					switch v.(type) {
					case []interface{}, map[string]interface{}:
						if b, err := json.Marshal(v); err == nil {
							fmt.Fprintf(w, "%s\t%s\n", strings.ToUpper(k), string(b))
							continue
						}
					}
				}
				fmt.Fprintf(w, "%s\t%s\n", strings.ToUpper(k), formatCell(v))
			}
			for _, na := range nestedArrays {
				fmt.Fprintln(w, "\n"+strings.ToUpper(na.key)+":")
				printArrayAsTable(na.arr)
			}
		}
		w.Flush()
	default:
		fmt.Print(text)
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

	return <-ch
}
