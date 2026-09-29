package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
)

// sseServer serves handleSSE as user "u" and returns the stream's frames
// (blank-line separated blocks) on a channel.
func sseServer(t *testing.T, keepalive time.Duration) (frames <-chan string, session func() chan string) {
	t.Helper()
	setSSEKeepalive(keepalive)
	t.Cleanup(func() { setSSEKeepalive(config.DefaultSSEKeepalive) })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleSSE(w, r.WithContext(context.WithValue(r.Context(), userCtxKey, "u")))
	}))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); srv.Close() })
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	out := make(chan string, 256)
	go func() {
		defer close(out)
		sc := bufio.NewScanner(resp.Body)
		var block []string
		for sc.Scan() {
			if line := sc.Text(); line != "" {
				block = append(block, line)
			} else if len(block) > 0 {
				out <- strings.Join(block, "\n")
				block = nil
			}
		}
	}()
	return out, func() chan string {
		sessionsMu.RLock()
		defer sessionsMu.RUnlock()
		for _, s := range sessions {
			return s.Event
		}
		return nil
	}
}

func TestSSEIdleStreamGetsKeepalives(t *testing.T) {
	frames, _ := sseServer(t, 50*time.Millisecond)
	if f := <-frames; !strings.HasPrefix(f, "event: endpoint") {
		t.Fatalf("first frame should announce the endpoint, got %q", f)
	}
	for i := 0; i < 3; i++ {
		select {
		case f := <-frames:
			if f != ": ping" {
				t.Fatalf("idle stream: want a ': ping' comment, got %q", f)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("no keepalive on an idle stream")
		}
	}
}

func TestSSEKeepalivesAndEventsNeverInterleave(t *testing.T) {
	frames, session := sseServer(t, 5*time.Millisecond)
	<-frames // endpoint
	ev := session()
	if ev == nil {
		t.Fatal("no session registered")
	}
	go func() {
		for i := 0; i < 50; i++ {
			ev <- fmt.Sprintf(`{"n":%d}`, i)
			time.Sleep(time.Millisecond)
		}
	}()
	msgs, pings := 0, 0
	for msgs < 50 {
		select {
		case f := <-frames:
			switch {
			case f == ": ping":
				pings++
			case strings.HasPrefix(f, "event: message\ndata: {"):
				msgs++
			default:
				t.Fatalf("malformed frame (interleaved write?): %q", f)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("stalled after %d messages", msgs)
		}
	}
	if pings == 0 {
		t.Error("expected keepalives among the events")
	}
}

func TestSSEKeepaliveConfig(t *testing.T) {
	var c config.DaemonConfig
	if got := c.SSEKeepalive(); got != config.DefaultSSEKeepalive {
		t.Errorf("default: got %v", got)
	}
	c.Server.SSEKeepaliveSeconds = 7
	if got := c.SSEKeepalive(); got != 7*time.Second {
		t.Errorf("configured: got %v", got)
	}
}

// server.* keys keep their running values on a reload; a changed keepalive is
// reported as needing a restart, like the port and TLS.
func TestReloadReportsKeepaliveChangeAsRestartOnly(t *testing.T) {
	var running, onDisk Config
	if got := restartOnlyChanges(running, onDisk); len(got) != 0 {
		t.Fatalf("unchanged config reported %v", got)
	}
	onDisk.Server.SSEKeepaliveSeconds = 30
	got := restartOnlyChanges(running, onDisk)
	if len(got) != 1 || !strings.HasPrefix(got[0], "server.sse_keepalive_seconds") {
		t.Errorf("got %v", got)
	}
}
