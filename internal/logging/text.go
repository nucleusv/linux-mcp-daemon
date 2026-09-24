package logging

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"sync"
)

// textHandler writes the human-readable format:
//
//	2026-09-24T10:41:26.138Z INFO  tool call user=alice tool=files/list ok=true
//
// time, level and message stay positional (no time=/level=/msg= keys); the
// attributes follow as key=value, quoted and escaped by slog's own text
// handler, so every record is still exactly one line. format: json keeps
// all keys for machine parsing.
type textHandler struct {
	mu    *sync.Mutex
	out   io.Writer
	opts  *slog.HandlerOptions
	attrs []slog.Attr
}

func newTextHandler(w io.Writer, opts *slog.HandlerOptions) *textHandler {
	return &textHandler{mu: &sync.Mutex{}, out: w, opts: opts}
}

func (h *textHandler) Enabled(_ context.Context, lv slog.Level) bool {
	return lv >= h.opts.Level.Level()
}

// dropBuiltins removes the keys the line prints positionally.
func dropBuiltins(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 && (a.Key == slog.TimeKey || a.Key == slog.LevelKey || a.Key == slog.MessageKey) {
		return slog.Attr{}
	}
	return a
}

func (h *textHandler) Handle(ctx context.Context, r slog.Record) error {
	var attrs bytes.Buffer
	inner := slog.Handler(slog.NewTextHandler(&attrs, &slog.HandlerOptions{ReplaceAttr: dropBuiltins}))
	if len(h.attrs) > 0 {
		inner = inner.WithAttrs(h.attrs)
	}
	if err := inner.Handle(ctx, r); err != nil {
		return err
	}

	var line bytes.Buffer
	line.WriteString(r.Time.Format("2006-01-02T15:04:05.000Z07:00"))
	line.WriteByte(' ')
	lv := r.Level.String()
	line.WriteString(lv)
	for i := len(lv); i < 5; i++ { // align messages: "INFO " / "ERROR"
		line.WriteByte(' ')
	}
	line.WriteByte(' ')
	line.WriteString(r.Message)
	if rest := bytes.TrimSpace(attrs.Bytes()); len(rest) > 0 {
		line.WriteByte(' ')
		line.Write(rest)
	}
	line.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(line.Bytes())
	return err
}

func (h *textHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	c.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &c
}

// WithGroup isn't used by mcpd; groups are flattened into the line.
func (h *textHandler) WithGroup(string) slog.Handler { return h }
