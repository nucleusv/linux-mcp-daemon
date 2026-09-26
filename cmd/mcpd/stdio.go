package main

// `mcpd stdio`: MCP over stdin/stdout, for clients that start a server as a
// local process (Claude Code, Claude Desktop, MCP Inspector, catalogs' build
// checks) - including over `ssh host mcpd stdio` or `docker exec -i`.
//
// The same RPC handlers as the network daemon, with a different trust model:
// there is no token and no MCP user, so the caller is an OS account, and
// nothing ever runs as root:
//   - started by an ordinary user, workers run as that user;
//   - started as root (docker exec, a catalog's container), --user NAME is
//     required and workers run as NAME - the master still never runs a tool;
//   - privileged: true is always refused (worker.NoRoot), mcp-sudo.yaml is
//     not read.
//
// stdout carries JSON-RPC and nothing else; logs go to stderr.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"github.com/nucleusv/linux-mcp-daemon/internal/logging"
	"github.com/nucleusv/linux-mcp-daemon/internal/rpc"
	"github.com/nucleusv/linux-mcp-daemon/internal/version"
	"github.com/nucleusv/linux-mcp-daemon/internal/worker"
)

const stdioUsage = "usage: mcpd stdio [--user NAME] [--config-dir DIR]"

// maxStdioMessage bounds one JSON-RPC line, like worker arguments.
const maxStdioMessage = 64 << 20

func runStdio(args []string) {
	var asUser, dir string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case (a == "--user" || a == "--config-dir") && i+1 < len(args):
			if a == "--user" {
				asUser = args[i+1]
			} else {
				dir = args[i+1]
			}
			i++
		case strings.HasPrefix(a, "--user="):
			asUser = strings.TrimPrefix(a, "--user=")
		case strings.HasPrefix(a, "--config-dir="):
			dir = strings.TrimPrefix(a, "--config-dir=")
		default:
			logging.Fatal("unknown argument ("+stdioUsage+")", "arg", a)
		}
	}

	username, err := stdioUser(asUser, os.Geteuid())
	if err != nil {
		logging.Fatal("cannot serve over stdio", "err", err)
	}

	// daemon.yaml is optional here: only the worker timeout, per-tool
	// timeouts and logging are used. Listeners, users and mcp-sudo.yaml
	// belong to the network daemon.
	timeout, tools := 30, config.ToolsConfig(nil)
	if dir == "" {
		dir = os.Getenv("MCPD_CONFIG_DIR")
	}
	explicit := dir != ""
	if !explicit {
		dir = configDir
	}
	if _, statErr := os.Stat(filepath.Join(dir, config.DaemonFile)); statErr == nil || explicit {
		c, _, err := config.LoadConfigDir(dir, false)
		if err != nil {
			logging.Fatal("cannot load config", "dir", dir, "err", err)
		}
		timeout, tools = c.Worker.TimeoutSeconds, c.Tools
		logging.Configure(c.Logging)
	}

	worker.NoRoot = true
	worker.Containerized = false

	h := rpc.NewRPCHandler(&config.SudoConfig{}, timeout, tools, &requestGroup, rpcCache, &cacheMu, resourceCache)
	logging.Info("serving MCP over stdio", "version", version.Version, "transport", "stdio", "user", username)
	serveStdio(h, username, os.Stdin, os.Stdout)
}

// stdioUser decides whose account the workers run as.
func stdioUser(asUser string, euid int) (string, error) {
	if euid == 0 {
		if asUser == "" {
			return "", errors.New("started as root: pass --user NAME (an unprivileged account) - mcpd stdio never runs tools as root")
		}
	} else {
		self, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("cannot look up the current user: %v", err)
		}
		if asUser != "" && asUser != self.Username {
			return "", fmt.Errorf("--user %s needs root; running as %s, workers can only run as %s", asUser, self.Username, self.Username)
		}
		asUser = self.Username
	}
	u, err := user.Lookup(asUser)
	if err != nil {
		return "", fmt.Errorf("no OS account %q: %v", asUser, err)
	}
	if u.Uid == "0" {
		return "", fmt.Errorf("%q is uid 0 (root) - mcpd stdio never runs tools as root", asUser)
	}
	return u.Username, nil
}

// serveStdio reads one JSON-RPC message per line from in and writes each
// response as one line to out, until in ends and every request in flight
// has answered. Requests run concurrently; a single goroutine writes, so
// responses never interleave.
func serveStdio(h *rpc.RPCHandler, username string, in io.Reader, out io.Writer) {
	session := &rpc.Session{ID: "stdio", User: username, Event: make(chan string, 16), Done: make(chan struct{})}

	written := make(chan struct{})
	go func() {
		defer close(written)
		w := bufio.NewWriter(out)
		for msg := range session.Event {
			w.WriteString(msg)
			w.WriteByte('\n')
			w.Flush()
		}
	}()

	var inflight sync.WaitGroup
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 64<<10), maxStdioMessage)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req rpc.JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			logging.Warn("invalid JSON-RPC message on stdin", "err", err)
			b, _ := json.Marshal(rpc.JSONRPCResponse{JSONRPC: "2.0", Error: map[string]interface{}{"code": -32700, "message": "Parse error"}})
			session.Event <- string(b)
			continue
		}
		logging.Debug("JSON-RPC request", "method", req.Method, "id", req.ID, "user", username, "session", session.ID)
		inflight.Add(1)
		go func() {
			defer inflight.Done()
			h.ProcessJSONRPC(session, req)
		}()
	}
	if err := sc.Err(); err != nil {
		logging.Error("reading stdin", "err", err)
	}
	inflight.Wait()
	close(session.Event)
	<-written
}
