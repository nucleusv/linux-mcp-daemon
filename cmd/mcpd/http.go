package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/rpc"
)

// newSessionID returns an opaque, random identifier for an SSE session.
// It must never be derived from the caller's bearer token, since session
// IDs travel in the /message?session_id=... URL and end up in access logs.
func newSessionID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand should never fail on Linux, but falling through with a
		// zeroed buffer would make every session ID on this path identical
		// (and therefore trivially guessable), so fall back to a value that
		// at least varies per call.
		log.Printf("crypto/rand.Read failed, falling back to a time-based session id: %v", err)
		binary.BigEndian.PutUint64(b, uint64(time.Now().UnixNano()))
	}
	return hex.EncodeToString(b)
}

type ctxKey int

const userCtxKey ctxKey = iota

// userFromContext returns the username the middleware authenticated for
// this request, avoiding a second authenticateRequest scan in handlers.
func userFromContext(ctx context.Context) (string, bool) {
	u, _ := ctx.Value(userCtxKey).(string)
	return u, u != ""
}

// statusRecorder captures the status code written by a handler so the
// access log middleware can report it after the handler returns.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Flush forwards to the underlying ResponseWriter's Flusher, if it has one.
// Without this, wrapping breaks streaming (SSE) responses: net/http buffers
// writes until the handler returns, since *statusRecorder wouldn't otherwise
// satisfy http.Flusher for handleSSE's w.(http.Flusher) type assertion.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// clientIP prefers X-Forwarded-For (set by a reverse proxy) over the raw
// socket address so access logs stay useful behind a proxy.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// loggingMiddleware writes one access-log line per request to stdout/stderr
// (captured by `docker logs`), in the style of an HTTP access log.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		user := "-"
		if username, ok := authenticateRequest(r); ok {
			user = username
			r = r.WithContext(context.WithValue(r.Context(), userCtxKey, username))
		}

		next.ServeHTTP(rec, r)

		log.Printf("[ACCESS] %s user=%s \"%s %s\" %d %s",
			clientIP(r), user, r.Method, r.URL.RequestURI(), rec.status, time.Since(start).Round(time.Millisecond))
	})
}

func authenticateRequest(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", false
	}
	providedToken := strings.TrimPrefix(authHeader, "Bearer ")
	for _, user := range daemonConfig.Users {
		if user.TokenHash != "" {
			// Salted-hash accounts (created or rotated via `linuxctl create|update
			// mcpd user`) - compare hash(salt+provided) against the stored hash
			// in constant time so a timing side-channel can't leak how many
			// hex characters matched.
			sum := sha256.Sum256([]byte(user.TokenSalt + providedToken))
			computedHash := hex.EncodeToString(sum[:])
			if subtle.ConstantTimeCompare([]byte(computedHash), []byte(user.TokenHash)) == 1 {
				return user.Username, true
			}
			continue
		}
		// Legacy plaintext accounts not yet migrated - still constant-time,
		// since a plaintext token is exactly as sensitive as a hash match.
		if subtle.ConstantTimeCompare([]byte(user.Token), []byte(providedToken)) == 1 && user.Token != "" {
			return user.Username, true
		}
	}
	return "", false
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	username, ok := userFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionUser := username

	sessionsMu.Lock()
	sessionCounter++
	uniqueSessionID := fmt.Sprintf("%s-%d", newSessionID(), sessionCounter)

	session := &rpc.Session{
		ID:    uniqueSessionID,
		User:  sessionUser,
		Event: make(chan string, 10),
	}
	sessions[uniqueSessionID] = session
	sessionsMu.Unlock()

	log.Printf("SSE connection established for user=%s session=%s", sessionUser, uniqueSessionID)

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
		log.Printf("SSE connection closed for user=%s session=%s", sessionUser, uniqueSessionID)
	}()

	for {
		select {
		case msg := <-session.Event:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-r.Context().Done():
			log.Printf("SSE connection closed for user=%s session=%s", username, uniqueSessionID)
			return
		}
	}
}

func handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := userFromContext(r.Context())
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

	// A session_id only routes a message to the right SSE stream - it must
	// never substitute for auth. Without this check, anyone who observes a
	// session_id (e.g. in logs, which include it in POST /message URLs) could
	// use their own valid token to execute tool calls under the session
	// owner's privileges instead of their own.
	if session.User != username {
		log.Printf("[SECURITY] user=%s attempted to use session=%s owned by user=%s", username, sessionID, session.User)
		http.Error(w, "Forbidden", http.StatusForbidden)
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

	log.Printf("Received JSON-RPC method=%s for user=%s session=%s", req.Method, username, sessionID)

	go rpcHandler.ProcessJSONRPC(session, req)

	w.WriteHeader(http.StatusAccepted)
}
