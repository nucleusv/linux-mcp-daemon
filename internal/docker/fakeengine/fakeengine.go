// Package fakeengine serves a scripted Docker Engine API on a throwaway unix
// socket, so the docker/* tools can be tested without Docker. Only tests
// import it; nothing in the daemon does.
//
// A route key is "METHOD /path" (no query string). Its value is the response
// body, optionally prefixed with a status code: "409 {\"message\":\"...\"}".
// Every request is recorded in order, including its query string, so a test
// can assert both what was called and what was *not* sent - which is how
// docker/manage's "never force, never v" rule is checked.
package fakeengine

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
)

type Request struct {
	Method string
	Path   string
	Query  string
	Body   string
}

type Engine struct {
	Socket string

	mu       sync.Mutex
	requests []Request
	routes   map[string]string
	delays   map[string]time.Duration
	dir      string
	srv      *http.Server
}

// Delay makes "METHOD /path" hang this long before answering, for the timeout
// paths - an exec or log read the client gives up on.
func (e *Engine) Delay(method, path string, d time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.delays == nil {
		e.delays = map[string]time.Duration{}
	}
	e.delays[method+" "+path] = d
}

// New starts an engine on a fresh unix socket. Close it when done.
func New(routes map[string]string) (*Engine, error) {
	// Not t.TempDir()/os.TempDir() nesting: a unix socket address is capped
	// at ~104 bytes, and a long path fails as "bind: invalid argument".
	dir, err := os.MkdirTemp("", "mcpd")
	if err != nil {
		return nil, err
	}
	e := &Engine{Socket: filepath.Join(dir, "d.sock"), routes: routes, dir: dir}
	ln, err := net.Listen("unix", e.Socket)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	e.srv = &http.Server{Handler: e}
	go func() { _ = e.srv.Serve(ln) }()
	return e, nil
}

func (e *Engine) Close() {
	_ = e.srv.Close()
	_ = os.RemoveAll(e.dir)
}

// Client is an Engine API client bound to this fake.
func (e *Engine) Client() *docker.Client { return docker.New(e.Socket, 5*time.Second) }

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	e.mu.Lock()
	e.requests = append(e.requests, Request{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(body)})
	answer, ok := e.routes[r.Method+" "+r.URL.Path]
	delay := e.delays[r.Method+" "+r.URL.Path]
	e.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"message":"page not found: %s %s"}`, r.Method, r.URL.Path)
		return
	}
	status := http.StatusOK
	if len(answer) > 4 && answer[3] == ' ' {
		if n, err := strconv.Atoi(answer[:3]); err == nil {
			status, answer = n, answer[4:]
		}
	}
	w.WriteHeader(status)
	_, _ = io.WriteString(w, answer)
}

// Requests returns every request served, in order.
func (e *Engine) Requests() []Request {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]Request{}, e.requests...)
}

// Called reports whether "METHOD /path" was requested, and its query string.
func (e *Engine) Called(method, path string) (string, bool) {
	for _, r := range e.Requests() {
		if r.Method == method && r.Path == path {
			return r.Query, true
		}
	}
	return "", false
}

// Frame builds one Docker stream frame: 1 = stdout, 2 = stderr. Used to feed
// the log and exec endpoints, whose bodies are multiplexed.
func Frame(typ byte, payload string) string {
	h := []byte{typ, 0, 0, 0, 0, 0, 0, 0}
	n := len(payload)
	h[4] = byte(n >> 24)
	h[5] = byte(n >> 16)
	h[6] = byte(n >> 8)
	h[7] = byte(n)
	return string(h) + payload
}

// Inspect is a minimal container inspect body: enough for docker.Resolve to
// map a name or ID prefix to one canonical container.
func Inspect(id, name string) string {
	return fmt.Sprintf(`{"Id":%q,"Name":"/%s"}`, id, strings.TrimPrefix(name, "/"))
}
