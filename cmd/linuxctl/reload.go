package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// reloadDaemon asks the running mcpd to re-read its config files after
// linuxctl changed them (daemon/reload-config). It never fails the command:
// the files are already written, so on any problem it says how to apply
// them by hand instead.
func reloadDaemon() {
	manual := "Apply by hand: linuxctl reload daemon (or restart mcpd; Kubernetes dev setup: scripts/deploy.sh)"

	authToken := *token
	if authToken == "" {
		authToken = os.Getenv("MCP_TOKEN")
	}
	if authToken == "" {
		fmt.Printf("\nNot applied yet: no MCP_TOKEN to call mcpd with.\n%s\n", manual)
		return
	}

	fmt.Printf("\nAsking mcpd at %s to reload its config...\n", *serverURL)
	if err := connect(authToken, "mcpd"); err != nil {
		fmt.Printf("Not applied yet: %v\n%s\n", err, manual)
		return
	}
	resp, err := tryCallMethod(authToken, nextID(), "tools/call", map[string]interface{}{
		"name":      "daemon/reload-config",
		"arguments": map[string]interface{}{},
	}, 30*time.Second)
	if err != nil {
		fmt.Printf("Not applied yet: %v\n%s\n", err, manual)
		return
	}
	if resp.Error != nil {
		fmt.Printf("Not applied yet: %s\n%s\n", resp.Error.Message, manual)
		return
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	_ = json.Unmarshal(resp.Result, &result)
	var text strings.Builder
	for _, c := range result.Content {
		text.WriteString(c.Text)
	}
	if result.IsError {
		fmt.Printf("Not applied: %s\n", strings.TrimSpace(text.String()))
		if strings.Contains(text.String(), "not authorized") {
			fmt.Println("Your token's user needs daemon/reload-config granted in mcp-sudo.yaml.")
		}
		fmt.Println(manual)
		return
	}
	fmt.Print(text.String())
	if strings.Contains(text.String(), "No changes.") {
		fmt.Printf("mcpd at %s saw no change - is it reading a different config than --config-path?\n", *serverURL)
	}
}
