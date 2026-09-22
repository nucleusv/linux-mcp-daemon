package main

import (
	"encoding/json"
	"log"
)

func processJSONRPC(session *Session, req JSONRPCRequest) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		handleInitialize(&resp)
	case "notifications/initialized":
		return
	case "ping":
		// Ping is just an empty map, handled implicitly
	case "resources/list":
		handleResourcesList(&resp)
	case "resources/templates/list":
		handleResourcesTemplatesList(&resp)
	case "resources/read":
		handleResourcesRead(session, req, &resp)
	case "tools/list":
		handleToolsList(session, &resp)
	case "tools/call":
		handleToolsCall(session, req, &resp)
	default:
		resp.Error = map[string]interface{}{"code": -32601, "message": "Method not found"}
	}
	respBytes, _ := json.Marshal(resp)
	log.Printf("Sending response for %s: %s", req.Method, string(respBytes))
	session.Event <- string(respBytes)
}

func handleInitialize(resp *JSONRPCResponse) {
	resp.Result = map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools":     map[string]interface{}{},
			"resources": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "linux-mcp-daemon",
			"version": "1.0.0",
		},
	}
}


