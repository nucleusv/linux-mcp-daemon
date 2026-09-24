// Package logging is mcpd's leveled, structured log (log/slog), configured
// by daemon.yaml's logging: block and changeable at runtime through
// daemon/reload-config.
//
// Two kinds of lines ignore the level: Access (one per HTTP request, when
// access_log is on) and Audit (every call that changes the host or the
// daemon's config) - turning detail down must never hide who changed what.
//
// Nothing here may log a token, a tool's output or unredacted arguments:
// callers pass arguments through rpc.redactArgs and output by size only.
package logging

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// Config is daemon.yaml's logging: block.
type Config struct {
	Level     string `yaml:"level"`      // error | warn | info | debug (default info)
	Format    string `yaml:"format"`     // text | json (default text)
	AccessLog *bool  `yaml:"access_log"` // one line per HTTP request (default true)
}

// Validate reports a level or format mcpd doesn't know.
func (c Config) Validate() error {
	if _, err := parseLevel(c.Level); err != nil {
		return err
	}
	switch strings.ToLower(c.Format) {
	case "", "text", "json":
		return nil
	}
	return fmt.Errorf("logging.format %q: must be text or json", c.Format)
}

func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	}
	return 0, fmt.Errorf("logging.level %q: must be error, warn, info or debug", s)
}

var (
	level    slog.LevelVar
	accessOn atomic.Bool
	format   atomic.Value // string
	output   io.Writer    = os.Stderr
	handler  atomic.Pointer[slog.Handler]
)

func init() {
	accessOn.Store(true)
	format.Store("")
	Configure(Config{})
}

// Configure applies a logging config; safe to call while logging (reload).
// It also routes the standard log package, used by code that predates
// slog, through the same handler at info level.
func Configure(c Config) {
	lv, err := parseLevel(c.Level)
	if err != nil {
		lv = slog.LevelInfo
	}
	level.Set(lv)
	accessOn.Store(c.AccessLog == nil || *c.AccessLog)
	f := strings.ToLower(c.Format)
	if prev, _ := format.Load().(string); prev == f && handler.Load() != nil {
		return
	}
	format.Store(f)
	opts := &slog.HandlerOptions{Level: &level}
	var h slog.Handler
	if f == "json" {
		h = slog.NewJSONHandler(output, opts)
	} else {
		h = newTextHandler(output, opts)
	}
	handler.Store(&h)
	slog.SetDefault(slog.New(h))
	log.SetFlags(0) // slog stamps the time itself
}

// SetOutput redirects logging (tests).
func SetOutput(w io.Writer, c Config) {
	output = w
	format.Store("\x00") // force a new handler
	Configure(c)
}

func Debug(msg string, args ...any) { slog.Debug(msg, args...) }
func Info(msg string, args ...any)  { slog.Info(msg, args...) }
func Warn(msg string, args ...any)  { slog.Warn(msg, args...) }
func Error(msg string, args ...any) { slog.Error(msg, args...) }

// Enabled reports whether lv would be logged (to skip building costly
// attributes).
func Enabled(lv slog.Level) bool { return lv >= level.Level() }

// Access logs one HTTP request, whatever the level, if access_log is on.
func Access(args ...any) {
	if accessOn.Load() {
		always(slog.LevelInfo, "access", args...)
	}
}

// Audit logs a change to the host or to mcpd's config, whatever the level.
func Audit(msg string, args ...any) {
	always(slog.LevelInfo, msg, append([]any{"audit", true}, args...)...)
}

// always writes a record bypassing the level check.
func always(lv slog.Level, msg string, args ...any) {
	h := *handler.Load()
	r := slog.NewRecord(time.Now(), lv, msg, 0)
	r.Add(args...)
	_ = h.Handle(context.Background(), r)
}

// Fatal logs why mcpd can't run and exits. Like Audit it ignores the
// level: a daemon that stops must always say why.
func Fatal(msg string, args ...any) {
	always(slog.LevelError, msg, args...)
	os.Exit(1)
}
