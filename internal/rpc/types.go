package rpc

import (
	"encoding/json"
	"sync"
	"time"
)

type Session struct {
	ID    string
	User  string
	Event chan string
	// Done is closed by Close to end the SSE stream from the server side
	// (e.g. the user was removed by a config reload).
	Done      chan struct{}
	closeOnce sync.Once
}

// Close ends the session's SSE stream. Safe to call more than once.
func (s *Session) Close() {
	s.closeOnce.Do(func() {
		if s.Done != nil {
			close(s.Done)
		}
	})
}

// MCP JSON-RPC Structures
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError,omitempty"`
}

type CacheEntry struct {
	Result    string
	ExpiresAt time.Time
}
