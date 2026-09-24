package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/nucleusv/linux-mcp-daemon/internal/auth"
	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"github.com/nucleusv/linux-mcp-daemon/internal/rpc"
)

func mustDaemon(t *testing.T, doc string) Config {
	t.Helper()
	c, err := config.ParseDaemonConfig([]byte(doc), true)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func mustSudo(t *testing.T, doc string) *config.SudoConfig {
	t.Helper()
	c, err := config.ParseSudoConfig([]byte(doc), true)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestConfigChanges(t *testing.T) {
	prev := mustDaemon(t, `users:
  - {username: alice, token_salt: s1, token_hash: h1}
  - {username: bob, token_salt: s2, token_hash: h2}
  - {username: carol, token_salt: s3, token_hash: h3}
`)
	next := mustDaemon(t, `users:
  - {username: alice, token_salt: s1, token_hash: h1}
  - {username: bob, token_salt: s9, token_hash: h9}
  - {username: dave, token_salt: s4, token_hash: h4}
`)
	prevSudo := mustSudo(t, `users:
  alice:
    privileged:
      tools:
        files/read: {allowed: true, paths: [/home/alice]}
        disks/usage: {allowed: true, paths: ["/"]}
`)
	nextSudo := mustSudo(t, `users:
  alice:
    privileged:
      tools:
        files/read: {allowed: true, paths: [/]}
        files/chmod: {allowed: true, paths: [/var/www]}
      resources:
        file://: [""]
`)

	changes, closeUsers := configChanges(prev, next, prevSudo, nextSudo)
	want := []string{
		"user bob: token changed",
		"user carol: removed",
		"user dave: added",
		"grants alice: - disks/usage",
		"grants alice: + files/chmod (root; paths [/var/www])",
		"grants alice: ~ files/read (root; paths [/])",
		`grants alice: + resource file:// [""]`,
	}
	if !reflect.DeepEqual(changes, want) {
		t.Errorf("changes:\n%s\nwant:\n%s", strings.Join(changes, "\n"), strings.Join(want, "\n"))
	}
	if !reflect.DeepEqual(closeUsers, map[string]bool{"bob": true, "carol": true}) {
		t.Errorf("closeUsers = %v", closeUsers)
	}
	for _, c := range changes {
		for _, secret := range []string{"s9", "h9", "h1", "s1"} {
			if strings.Contains(c, secret) {
				t.Errorf("change %q leaks a token salt/hash", c)
			}
		}
	}

	if c, _ := configChanges(prev, prev, prevSudo, prevSudo); len(c) != 0 {
		t.Errorf("identical configs reported changes: %v", c)
	}
}

func TestRestartOnlyChanges(t *testing.T) {
	a := mustDaemon(t, "server: {port: 9091}\n")
	b := mustDaemon(t, "server: {port: 9092}\nworker: {containerized: true}\n")
	got := restartOnlyChanges(a, b)
	want := []string{"server.port: 9091 -> 9092", "worker.containerized: false -> true"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// setupConfigDir makes a configs/ dir in a temp working directory and
// points the daemon's globals at it, as main() does.
func setupConfigDir(t *testing.T, users, sudo string) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Mkdir(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(name, data string) {
		if err := os.WriteFile(filepath.Join(configDir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(config.DaemonFile, "server: {port: 9091}\n")
	write(config.UsersFile, users)
	write(config.SudoFile, sudo)

	var err error
	daemonConfig, usersPath, err = loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	sudoCfg, err := config.LoadSudoConfig(sudoConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	limiterManager.Store(auth.NewLimiterManager(daemonConfig.RateLimits.DefaultRPS, daemonConfig.RateLimits.DefaultBurst))
	rpcHandler = rpc.NewRPCHandler(sudoCfg, 30, nil, &requestGroup, rpcCache, &cacheMu, resourceCache)
	rpcHandler.ReloadConfig = reloadConfig
	return dir
}

func userEntry(name, token string) string {
	salt := "salt-" + name
	sum := sha256.Sum256([]byte(salt + token))
	return fmt.Sprintf("  - {username: %s, token_salt: %s, token_hash: %s}\n", name, salt, hex.EncodeToString(sum[:]))
}

func authAs(token string) (string, bool) {
	r := httptest.NewRequest("GET", "/sse", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	return authenticateRequest(r)
}

func TestReloadConfig(t *testing.T) {
	setupConfigDir(t, "users:\n"+userEntry("root", "tok-root"), "users: {}\n")

	if _, ok := authAs("tok-new"); ok {
		t.Fatal("unknown token authenticated")
	}

	// Add a second token for root's replacement user and grant a tool.
	os.WriteFile(filepath.Join(configDir, config.UsersFile), []byte("users:\n"+userEntry("root", "tok-new")), 0600)
	os.WriteFile(sudoConfigPath(), []byte("users:\n  root:\n    privileged:\n      tools:\n        daemon/reload-config: {allowed: true}\n"), 0600)

	// An open session of root must be closed: its token changed.
	s := &rpc.Session{ID: "s1", User: "root", Event: make(chan string), Done: make(chan struct{})}
	sessionsMu.Lock()
	sessions[s.ID] = s
	sessionsMu.Unlock()
	defer func() { sessionsMu.Lock(); delete(sessions, s.ID); sessionsMu.Unlock() }()

	// Reload while requests authenticate concurrently (run with -race).
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				authAs("tok-root")
				authAs("tok-new")
				rpcHandler.Sudo().CanRunAsRoot("root", "daemon/reload-config")
			}
		}()
	}
	out, err := reloadConfig("tester")
	wg.Wait()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"user root: token changed", "grants root: + daemon/reload-config (root)", "Closed 1 session(s)"} {
		if !strings.Contains(out, want) {
			t.Errorf("reload output lacks %q:\n%s", want, out)
		}
	}
	select {
	case <-s.Done:
	default:
		t.Error("session of a user whose token changed is still open")
	}
	if _, ok := authAs("tok-root"); ok {
		t.Error("old token still authenticates after reload")
	}
	if u, ok := authAs("tok-new"); !ok || u != "root" {
		t.Error("new token doesn't authenticate after reload")
	}
	if !rpcHandler.Sudo().CanRunAsRoot("root", "daemon/reload-config") {
		t.Error("new grant not in effect after reload")
	}

	// An invalid file (misspelled key) is rejected and changes nothing.
	os.WriteFile(sudoConfigPath(), []byte("users:\n  root:\n    privileged:\n      tools:\n        files/read: {allowed: true, path: [/]}\n"), 0600)
	if _, err := reloadConfig("tester"); err == nil || !strings.Contains(err.Error(), "stays in effect") {
		t.Errorf("invalid config: err = %v", err)
	}
	if !rpcHandler.Sudo().CanRunAsRoot("root", "daemon/reload-config") || rpcHandler.Sudo().CanRunAsRoot("root", "files/read") {
		t.Error("a rejected reload changed the rules in effect")
	}
	if _, ok := authAs("tok-new"); !ok {
		t.Error("a rejected reload changed the users")
	}
}
