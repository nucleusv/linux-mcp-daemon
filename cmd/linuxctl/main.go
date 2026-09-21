package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	serverURL := flag.String("server", "http://localhost:9090", "The URL of the mcpd server")
	token := flag.String("token", "", "Bearer token for authentication")
	flag.Parse()

	// Ensure token is provided, or get from env
	authToken := *token
	if authToken == "" {
		authToken = os.Getenv("MCP_TOKEN")
		if authToken == "" {
			fmt.Println("Error: Authentication token is required.")
			fmt.Println("Provide it via -token flag or MCP_TOKEN environment variable.")
			os.Exit(1)
		}
	}

	command := flag.Arg(0)
	if command == "" {
		fmt.Println("Usage: linuxctl [options] <command>")
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("\nAvailable commands:")
		fmt.Println("  ping    - Test connection to the mcpd daemon")
		os.Exit(1)
	}

	// Basic command routing logic for the client
	switch command {
	case "ping":
		// Example: simply trying to connect to the SSE endpoint or a hypothetical status endpoint
		req, err := http.NewRequest("GET", *serverURL+"/sse", nil)
		if err != nil {
			fmt.Printf("Failed to create request: %v\n", err)
			os.Exit(1)
		}
		
		req.Header.Set("Authorization", "Bearer "+authToken)
		
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Failed to connect to daemon: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Println("Successfully connected to mcpd daemon!")
			// Here you could read from SSE if desired, but for ping we just exit
		} else {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("Daemon returned status: %s\n%s\n", resp.Status, string(body))
		}
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
