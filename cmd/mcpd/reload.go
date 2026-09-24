package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/nucleusv/linux-mcp-daemon/internal/auth"
	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"github.com/nucleusv/linux-mcp-daemon/internal/logging"
)

// reloadMu serializes reloads, so two concurrent daemon/reload-config
// calls can't interleave their read-validate-swap steps.
var reloadMu sync.Mutex

// reloadConfig re-reads daemon.yaml, users.yaml and mcp-sudo.yaml and swaps them into
// the running daemon (daemon/reload-config). Both files are parsed and
// validated before anything changes: if either is invalid, the daemon
// keeps running on the config it already has. It only reads the files.
func reloadConfig(byUser string) (string, error) {
	reloadMu.Lock()
	defer reloadMu.Unlock()

	next, nextUsersPath, err := config.LoadConfigDir(configDir, true)
	if err != nil {
		logging.Warn("config reload rejected, keeping the current config", "user", byUser, "err", err)
		return "", fmt.Errorf("config not reloaded, the current one stays in effect: %w", err)
	}
	nextSudo, err := config.LoadSudoConfigStrict(sudoConfigPath())
	if err != nil {
		logging.Warn("config reload rejected, keeping the current config", "user", byUser, "err", err)
		return "", fmt.Errorf("config not reloaded, the current one stays in effect: %w", err)
	}

	prevSudo := rpcHandler.Sudo()

	cfgMu.Lock()
	prev := daemonConfig
	restartOnly := restartOnlyChanges(prev, next)
	// Compared under cfgMu: checkAndPinUID updates user entries in place.
	changes, closeUsers := configChanges(prev, next, prevSudo, nextSudo)
	// Settings that belong to the running listeners and worker setup keep
	// their current values until a restart.
	next.Server = prev.Server
	next.Worker.Containerized = prev.Worker.Containerized
	daemonConfig = next
	usersPath = nextUsersPath
	// Users and grants switch together, never one without the other.
	limiterManager.Store(auth.NewLimiterManager(next.RateLimits.DefaultRPS, next.RateLimits.DefaultBurst))
	rpcHandler.Reconfigure(nextSudo, next.Worker.TimeoutSeconds, next.Tools)
	logging.Configure(next.Logging)
	cfgMu.Unlock()

	closed := closeSessions(closeUsers)

	var b strings.Builder
	if daemonPath := filepath.Join(configDir, config.DaemonFile); nextUsersPath == daemonPath {
		fmt.Fprintf(&b, "Reloaded %s (including its users) and %s.\n", daemonPath, sudoConfigPath())
	} else {
		fmt.Fprintf(&b, "Reloaded %s, %s and %s.\n", daemonPath, nextUsersPath, sudoConfigPath())
	}
	if len(changes) == 0 {
		b.WriteString("No changes.\n")
	} else {
		b.WriteString("Changes:\n")
		for _, c := range changes {
			fmt.Fprintf(&b, "  %s\n", c)
		}
	}
	if closed > 0 {
		fmt.Fprintf(&b, "Closed %d session(s) of removed users or users whose token changed.\n", closed)
	}
	if len(restartOnly) > 0 {
		b.WriteString("Need a restart to take effect (not applied):\n")
		for _, c := range restartOnly {
			fmt.Fprintf(&b, "  %s\n", c)
		}
	}

	if w := legacyUsersWarning(next, nextUsersPath); w != "" {
		fmt.Fprintf(&b, "Note: %s\n", w)
	}

	logging.Audit("config reloaded", "user", byUser, "changes", len(changes), "detail", strings.Join(changes, "; "))
	return b.String(), nil
}

// configChanges lists what differs between two configs, and which users'
// open sessions must end (removed, or their token changed). It never
// includes token values or hashes.
func configChanges(prev, next Config, prevSudo, nextSudo *config.SudoConfig) (changes []string, closeUsers map[string]bool) {
	closeUsers = map[string]bool{}

	type tok struct{ token, salt, hash string }
	prevUsers := map[string]tok{}
	for _, u := range prev.Users {
		prevUsers[u.Username] = tok{u.Token, u.TokenSalt, u.TokenHash}
	}
	nextUsers := map[string]tok{}
	for _, u := range next.Users {
		nextUsers[u.Username] = tok{u.Token, u.TokenSalt, u.TokenHash}
	}
	for _, name := range sortedKeys(prevUsers, nextUsers) {
		p, inPrev := prevUsers[name]
		n, inNext := nextUsers[name]
		switch {
		case !inNext:
			changes = append(changes, fmt.Sprintf("user %s: removed", name))
			closeUsers[name] = true
		case !inPrev:
			changes = append(changes, fmt.Sprintf("user %s: added", name))
		case p != n:
			changes = append(changes, fmt.Sprintf("user %s: token changed", name))
			closeUsers[name] = true
		}
	}

	prevGrants, nextGrants := map[string]config.UserSudo{}, map[string]config.UserSudo{}
	if prevSudo != nil {
		prevGrants = prevSudo.Users
	}
	if nextSudo != nil {
		nextGrants = nextSudo.Users
	}
	for _, name := range sortedKeys(prevGrants, nextGrants) {
		p, n := prevGrants[name].Privileged, nextGrants[name].Privileged
		for _, tool := range sortedKeys(p.Tools, n.Tools) {
			pt, inPrev := p.Tools[tool]
			nt, inNext := n.Tools[tool]
			switch {
			case !inNext:
				changes = append(changes, fmt.Sprintf("grants %s: - %s", name, tool))
			case !inPrev:
				changes = append(changes, fmt.Sprintf("grants %s: + %s%s", name, tool, grantDetail(nt)))
			case !reflect.DeepEqual(pt, nt):
				changes = append(changes, fmt.Sprintf("grants %s: ~ %s%s", name, tool, grantDetail(nt)))
			}
		}
		for _, res := range sortedKeys(p.Resources, n.Resources) {
			pr, inPrev := p.Resources[res]
			nr, inNext := n.Resources[res]
			switch {
			case !inNext:
				changes = append(changes, fmt.Sprintf("grants %s: - resource %s", name, res))
			case !inPrev:
				changes = append(changes, fmt.Sprintf("grants %s: + resource %s %q", name, res, nr))
			case !reflect.DeepEqual(pr, nr):
				changes = append(changes, fmt.Sprintf("grants %s: ~ resource %s %q", name, res, nr))
			}
		}
	}

	if prev.RateLimits != next.RateLimits {
		changes = append(changes, fmt.Sprintf("rate limits: %g rps, burst %d", next.RateLimits.DefaultRPS, next.RateLimits.DefaultBurst))
	}
	if !reflect.DeepEqual(prev.Logging, next.Logging) {
		changes = append(changes, fmt.Sprintf("logging: level %s, format %s, access log %t",
			orDefault(next.Logging.Level, "info"), orDefault(next.Logging.Format, "text"), next.Logging.AccessLog == nil || *next.Logging.AccessLog))
	}
	if prev.Worker.TimeoutSeconds != next.Worker.TimeoutSeconds || !reflect.DeepEqual(prev.Tools, next.Tools) {
		changes = append(changes, fmt.Sprintf("tool timeouts: default %ds", next.Worker.TimeoutSeconds))
	}
	return changes, closeUsers
}

// grantDetail shows a grant's settings in one line, e.g.
// " (root; paths [/var/www])".
func grantDetail(t config.ToolPrivilege) string {
	var parts []string
	if t.Allowed {
		parts = append(parts, "root")
	} else {
		parts = append(parts, "no root")
	}
	if len(t.Paths) > 0 {
		parts = append(parts, fmt.Sprintf("paths %v", t.Paths))
	}
	if t.Network != nil {
		parts = append(parts, "network policy")
	}
	if t.Sysctl != nil && len(t.Sysctl.WriteKeys) > 0 {
		parts = append(parts, fmt.Sprintf("sysctl write_keys %v", t.Sysctl.WriteKeys))
	}
	return " (" + strings.Join(parts, "; ") + ")"
}

// restartOnlyChanges lists settings changed on disk that only a restart
// applies: the listeners and the worker's namespace handling.
func restartOnlyChanges(running, onDisk Config) []string {
	var out []string
	if onDisk.Server.Port != running.Server.Port {
		out = append(out, fmt.Sprintf("server.port: %d -> %d", running.Server.Port, onDisk.Server.Port))
	}
	if !reflect.DeepEqual(onDisk.Server.TLS, running.Server.TLS) {
		out = append(out, "server.tls")
	}
	if !reflect.DeepEqual(onDisk.Server.HTTP, running.Server.HTTP) {
		out = append(out, "server.http")
	}
	if onDisk.Worker.Containerized != running.Worker.Containerized {
		out = append(out, fmt.Sprintf("worker.containerized: %t -> %t", running.Worker.Containerized, onDisk.Worker.Containerized))
	}
	return out
}

// closeSessions ends the SSE streams of the given users and returns how
// many it closed. Their tokens no longer authenticate, so the streams
// could carry nothing more anyway.
func closeSessions(users map[string]bool) int {
	if len(users) == 0 {
		return 0
	}
	n := 0
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	for _, s := range sessions {
		if users[s.User] {
			s.Close()
			n++
		}
	}
	return n
}

func sortedKeys[V any](maps ...map[string]V) []string {
	set := map[string]bool{}
	for _, m := range maps {
		for k := range m {
			set[k] = true
		}
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
