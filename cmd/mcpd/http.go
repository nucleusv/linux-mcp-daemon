package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/rpc"
)

func authenticateRequest(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", false
	}
	providedToken := strings.TrimPrefix(authHeader, "Bearer ")
	for _, user := range daemonConfig.Users {
		if user.Token == providedToken {
			return user.Username, true
		}
	}
	return "", false
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	username, ok := authenticateRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	sessionUser := username

	sessionsMu.Lock()
	sessionCounter++
	uniqueSessionID := fmt.Sprintf("%s-%d", token, sessionCounter)

	session := &rpc.Session{
		ID:    uniqueSessionID,
		User:  sessionUser,
		Event: make(chan string, 10),
	}
	sessions[uniqueSessionID] = session
	sessionsMu.Unlock()

	log.Printf("SSE connection established for user: %s (Session: %s)", sessionUser, uniqueSessionID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	fmt.Fprintf(w, "event: endpoint\ndata: /message?session_id=%s\n\n", uniqueSessionID)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	defer func() {
		sessionsMu.Lock()
		delete(sessions, uniqueSessionID)
		sessionsMu.Unlock()
		close(session.Event)
		log.Printf("SSE connection closed for user: %s (Session: %s)", sessionUser, uniqueSessionID)
	}()

	for {
		select {
		case msg := <-session.Event:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-r.Context().Done():
			log.Printf("SSE connection closed for user: %s", username)
			return
		}
	}
}

func handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := authenticateRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "Missing session_id in query parameters", http.StatusBadRequest)
		return
	}

	sessionsMu.RLock()
	session, exists := sessions[sessionID]
	sessionsMu.RUnlock()

	if !exists {
		http.Error(w, "No active SSE session", http.StatusBadRequest)
		return
	}

	if !limiterManager.Allow(username) {
		log.Printf("[THROTTLED] User %s exceeded rate limits", username)
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var req rpc.JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("Failed to unmarshal JSON-RPC: %v", err)
		http.Error(w, "Invalid JSON-RPC", http.StatusBadRequest)
		return
	}

	log.Printf("Received JSON-RPC method: %s for session %s", req.Method, sessionID)

	go rpcHandler.ProcessJSONRPC(session, req)

	w.WriteHeader(http.StatusAccepted)
}
