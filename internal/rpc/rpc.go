package rpc

import (
	"github.com/nucleusv/linux-mcp-daemon/internal/logging"

	"github.com/nucleusv/linux-mcp-daemon/internal/version"

	"encoding/json"
)

func (h *RPCHandler) ProcessJSONRPC(session *Session, req JSONRPCRequest) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	// A message without an id is a notification (notifications/initialized,
	// notifications/cancelled, ...): JSON-RPC forbids answering it. Over SSE
	// a stray answer went unnoticed; over stdio the client reads it as an
	// unexpected message. MCP never sends a request with a null id.
	if req.ID == nil {
		return
	}

	switch req.Method {
	case "initialize":
		h.HandleInitialize(&resp)
	case "ping":
		// Ping is just an empty map, handled implicitly
	case "resources/list":
		h.HandleResourcesList(&resp)
	case "resources/templates/list":
		h.HandleResourcesTemplatesList(&resp)
	case "resources/read":
		h.HandleResourcesRead(session, req, &resp)
	case "tools/list":
		h.HandleToolsList(session, &resp)
	case "tools/call":
		h.HandleToolsCall(session, req, &resp)
	default:
		resp.Error = map[string]interface{}{"code": -32601, "message": "Method not found"}
	}
	respBytes, _ := json.Marshal(resp)
	// Size only: response bodies are tool output - file contents read as
	// root, process environments, journal lines - and the daemon's own log
	// is readable by anyone in the adm/systemd-journal groups.
	logging.Debug("JSON-RPC response", "method", req.Method, "id", req.ID, "bytes", len(respBytes), "error", resp.Error != nil)
	session.Event <- string(respBytes)
}

func (h *RPCHandler) HandleInitialize(resp *JSONRPCResponse) {
	resp.Result = map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools":     map[string]interface{}{},
			"resources": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "linux-mcp-daemon",
			"version": version.Version,
		},
	}
}
