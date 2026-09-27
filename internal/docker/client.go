// Package docker talks to the Docker Engine API over its unix socket.
//
// The Engine API is plain HTTP/JSON on a unix socket, so this needs no
// dependency beyond net/http - the same "parse natively, don't wrap the
// CLI" rule the rest of this daemon follows. The `docker` binary is never
// invoked, and need not be installed.
package docker

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultSocket is Docker's usual socket path. The daemon passes the
// configured path in `_docker_socket`; this is the fallback so a worker run
// by hand still works.
const DefaultSocket = "/var/run/docker.sock"

// SocketPath is daemon.yaml's worker.docker_socket, set once at startup by
// cmd/mcpd/main.go (like worker.Containerized) and read by the master when it
// injects `_docker_socket` into a docker tool call. It lives here rather than
// in the RPC handler's settings because worker mode never loads daemon.yaml at
// all, so the value has to travel in the call arguments. Empty means
// DefaultSocket; changing it needs a restart, not a config reload.
var SocketPath string

// Client is a Docker Engine API client bound to one unix socket.
type Client struct {
	socket string
	dial   string
	http   *http.Client
}

// dialPath returns the path to actually connect to. It differs from the
// configured one in exactly one case: a containerized privileged worker.
// There, only the worker's *main thread* joined the host mount namespace -
// setns(2) is per-thread - while net/http dials from a goroutine of its own,
// on a thread still inside the container, where /var/run/docker.sock does not
// exist. (The symptom is confusing: os.Stat on the main thread finds the
// socket, the dial gets ENOENT.) /proc/1/root is the host's root from either
// namespace, so prefixing it makes the path thread-independent. Symlinks are
// resolved first because they are followed relative to the *dialing* thread's
// root: the host's /var/run -> /run would otherwise land in the container.
func dialPath(socket string) string {
	const hostRoot = "/proc/1/root"
	if os.Getenv("MCPD_HOST_ROOT") != "1" || strings.HasPrefix(socket, hostRoot+"/") {
		return socket
	}
	if resolved, err := filepath.EvalSymlinks(socket); err == nil {
		socket = resolved
	}
	return hostRoot + socket
}

// New returns a client for the socket at path (DefaultSocket when empty).
// The socket is not touched until the first request.
func New(socket string, timeout time.Duration) *Client {
	if socket == "" {
		socket = DefaultSocket
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	dial := dialPath(socket)
	return &Client{
		socket: socket,
		dial:   dial,
		http: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", dial)
				},
			},
		},
	}
}

// Socket is the path this client talks to.
func (c *Client) Socket() string { return c.socket }

// do performs one request and returns the response for the caller to read.
// A dial failure becomes an error naming the configured socket path and the
// likely cause, rather than a raw syscall error - the socket not being there
// is the single most common way these tools fail.
func (c *Client) do(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, "http://docker"+path, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, c.dialError(err)
	}
	return resp, nil
}

func (c *Client) dialError(err error) error {
	if _, statErr := os.Stat(c.dial); statErr != nil {
		if os.IsNotExist(statErr) {
			return fmt.Errorf("docker socket %s not found - is Docker installed and running on this host? (mcpd never installs it; set worker.docker_socket in daemon.yaml for a non-standard path)", c.socket)
		}
		return fmt.Errorf("docker socket %s is not usable: %v", c.socket, statErr)
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return fmt.Errorf("docker socket %s is not responding: %v", c.socket, opErr.Err)
	}
	return fmt.Errorf("docker socket %s: %v", c.socket, err)
}

// APIError is a non-2xx answer from the Engine API, with Docker's own
// message - which is more useful than anything this daemon could invent
// (e.g. the 409 telling you a container is still running).
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("docker API returned %d", e.Status)
	}
	return fmt.Sprintf("docker API returned %d: %s", e.Status, e.Message)
}

// apiError reads Docker's {"message": "..."} error body.
func apiError(resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var m struct {
		Message string `json:"message"`
	}
	msg := strings.TrimSpace(string(b))
	if json.Unmarshal(b, &m) == nil && m.Message != "" {
		msg = m.Message
	}
	return &APIError{Status: resp.StatusCode, Message: msg}
}

// GetJSON performs a GET and decodes the JSON body into out.
func (c *Client) GetJSON(path string, out interface{}) error {
	resp, err := c.do(http.MethodGet, path, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return apiError(resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// GetRaw performs a GET and returns the raw body (for endpoints whose
// answer is passed through unchanged, like full inspect JSON).
func (c *Client) GetRaw(path string) ([]byte, error) {
	resp, err := c.do(http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, apiError(resp)
	}
	return io.ReadAll(resp.Body)
}

// GetStream performs a GET and returns the undecoded body, for the
// multiplexed streams (logs, exec output). The caller closes it.
func (c *Client) GetStream(path string) (io.ReadCloser, error) {
	resp, err := c.do(http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		return nil, apiError(resp)
	}
	return resp.Body, nil
}

// PostJSON performs a POST with an optional JSON body, decoding the answer
// into out when out is non-nil.
func (c *Client) PostJSON(path string, in, out interface{}) error {
	var body io.Reader
	contentType := ""
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = strings.NewReader(string(b))
		contentType = "application/json"
	}
	resp, err := c.do(http.MethodPost, path, body, contentType)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return apiError(resp)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// PostStream performs a POST and returns the response body for streaming
// (exec start). The caller closes it.
func (c *Client) PostStream(path string, in interface{}) (io.ReadCloser, error) {
	var body io.Reader
	contentType := ""
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return nil, err
		}
		body = strings.NewReader(string(b))
		contentType = "application/json"
	}
	resp, err := c.do(http.MethodPost, path, body, contentType)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		return nil, apiError(resp)
	}
	return resp.Body, nil
}

// Delete performs a DELETE.
//
// Note what is deliberately missing: no variadic query parameters. The one
// caller (docker/manage remove) must not be able to add Docker's `force` or
// `v` - see FR-011. Removing a running container is kill-then-remove, two
// audited calls; anonymous volumes are never deleted here.
func (c *Client) Delete(path string) error {
	resp, err := c.do(http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return apiError(resp)
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	return nil
}

// frames walks Docker's multiplexed stream (used by logs and exec when the
// container has no TTY), calling emit for each frame's payload. Each frame is
// an 8-byte header - stream type, three zero bytes, then a big-endian uint32
// length - followed by that many payload bytes.
//
// A stream from a TTY container is not framed at all; it is emitted whole as
// stdout, which is what the caller wants to show either way.
func frames(r io.Reader, limit int64, emit func(typ byte, payload []byte)) error {
	raw, err := io.ReadAll(io.LimitReader(r, limit))
	if err != nil {
		return err
	}
	i := 0
	for i+8 <= len(raw) {
		typ := raw[i]
		if typ > 2 || raw[i+1] != 0 || raw[i+2] != 0 || raw[i+3] != 0 {
			emit(1, raw) // not a frame header: unframed (TTY) stream
			return nil
		}
		n := int(binary.BigEndian.Uint32(raw[i+4 : i+8]))
		i += 8
		if n < 0 || i+n > len(raw) {
			n = len(raw) - i // truncated final frame (hit the limit)
		}
		emit(typ, raw[i:i+n])
		i += n
	}
	return nil
}

// Demux splits a multiplexed stream into stdout and stderr. This is what exec
// wants: the two are reported separately, like any command's output.
func Demux(r io.Reader, limit int64) (stdout, stderr string, err error) {
	var out, errb strings.Builder
	err = frames(r, limit, func(typ byte, p []byte) {
		if typ == 2 {
			errb.Write(p)
		} else {
			out.Write(p)
		}
	})
	return out.String(), errb.String(), err
}

// DemuxCombined flattens a multiplexed stream in frame order. This is what
// logs want: stdout and stderr interleaved as Docker recorded them, which is
// what `docker logs` shows. Splitting them would scramble the chronology.
func DemuxCombined(r io.Reader, limit int64) (string, error) {
	var b strings.Builder
	err := frames(r, limit, func(_ byte, p []byte) { b.Write(p) })
	return b.String(), err
}

// Q builds a query string from pairs, skipping empty values.
func Q(pairs ...string) string {
	v := url.Values{}
	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i+1] != "" {
			v.Set(pairs[i], pairs[i+1])
		}
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}
