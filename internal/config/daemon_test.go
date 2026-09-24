package config

import (
	"os"
	"strings"
	"testing"
)

// Every config shipped in the repo must pass the strict parse that
// daemon/reload-config and linuxctl edit use.
func TestShippedConfigsParseStrictly(t *testing.T) {
	for _, dir := range []string{"../../configs", "../../packaging/configs", "../../packaging/configs-container"} {
		if _, err := LoadDaemonConfig(dir+"/daemon.yaml", true); err != nil {
			t.Errorf("daemon.yaml: %v", err)
		}
		if _, err := LoadSudoConfigStrict(dir + "/mcp-sudo.yaml"); err != nil {
			t.Errorf("mcp-sudo.yaml: %v", err)
		}
	}
}

func TestParseDaemonConfig(t *testing.T) {
	c, err := ParseDaemonConfig([]byte("users: []\n"), true)
	if err != nil {
		t.Fatal(err)
	}
	if c.Server.Port != 9091 || c.Worker.TimeoutSeconds != 30 {
		t.Errorf("defaults not applied: port %d, timeout %d", c.Server.Port, c.Worker.TimeoutSeconds)
	}
	if c.RateLimits.DefaultRPS != 50 || c.RateLimits.DefaultBurst != 100 {
		t.Errorf("rate limit defaults not applied: %+v", c.RateLimits)
	}

	bad := map[string]string{
		"unknown key":    "server:\n  prot: 9091\n",
		"no username":    "users:\n  - token_salt: s\n    token_hash: h\n",
		"duplicate user": "users:\n  - {username: a, token: x}\n  - {username: a, token: y}\n",
		"no token":       "users:\n  - username: a\n",
		"hash no salt":   "users:\n  - {username: a, token_hash: h}\n",
	}
	for name, doc := range bad {
		if _, err := ParseDaemonConfig([]byte(doc), true); err == nil {
			t.Errorf("%s: accepted %q", name, doc)
		}
	}
	// Lenient mode (daemon startup) only tolerates unknown keys.
	if _, err := ParseDaemonConfig([]byte(bad["unknown key"]), false); err != nil {
		t.Errorf("lenient parse rejected an unknown key: %v", err)
	}
}

func TestParseSudoConfigStrict(t *testing.T) {
	typo := "users:\n  alice:\n    privileged:\n      tools:\n        files/read:\n          allowed: true\n          path: [/tmp]\n"
	_, err := ParseSudoConfig([]byte(typo), true)
	if err == nil || !strings.Contains(err.Error(), "path") {
		t.Errorf("strict parse should reject the misspelled `path:`, got %v", err)
	}
	cfg, err := ParseSudoConfig([]byte(typo), false)
	if err != nil {
		t.Fatalf("lenient parse: %v", err)
	}
	if got := cfg.Users["alice"].Privileged.Tools["files/read"].Paths; len(got) != 0 {
		t.Errorf("lenient parse should drop the unknown key, got paths %v", got)
	}
}

func TestLoadConfigDir(t *testing.T) {
	write := func(dir, name, data string) {
		if err := os.WriteFile(dir+"/"+name, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	user := "users:\n  - {username: a, token: x}\n"

	legacy := t.TempDir()
	write(legacy, DaemonFile, user)
	c, from, err := LoadConfigDir(legacy, true)
	if err != nil || from != legacy+"/"+DaemonFile || len(c.Users) != 1 {
		t.Errorf("legacy: users %v from %q, err %v", c.Users, from, err)
	}

	split := t.TempDir()
	write(split, DaemonFile, "server: {port: 9091}\n")
	write(split, UsersFile, user)
	c, from, err = LoadConfigDir(split, true)
	if err != nil || from != split+"/"+UsersFile || len(c.Users) != 1 {
		t.Errorf("users.yaml: users %v from %q, err %v", c.Users, from, err)
	}

	both := t.TempDir()
	write(both, DaemonFile, user)
	write(both, UsersFile, user)
	if _, _, err := LoadConfigDir(both, true); err == nil || !strings.Contains(err.Error(), "both") {
		t.Errorf("users in both files: err = %v", err)
	}
}

func TestPathsOnlyOnPathTools(t *testing.T) {
	doc := "users:\n  a:\n    privileged:\n      tools:\n        disks/health: {allowed: true, paths: [/dev/sda]}\n"
	if _, err := ParseSudoConfig([]byte(doc), true); err == nil || !strings.Contains(err.Error(), "paths has no effect") {
		t.Errorf("strict: paths on disks/health accepted, err = %v", err)
	}
	if _, err := ParseSudoConfig([]byte(doc), false); err != nil {
		t.Errorf("lenient (startup) must still load: %v", err)
	}
	ok := "users:\n  a:\n    privileged:\n      tools:\n        disks/usage: {allowed: true, paths: [/var]}\n"
	if _, err := ParseSudoConfig([]byte(ok), true); err != nil {
		t.Errorf("paths on disks/usage rejected: %v", err)
	}
}

func TestResourceGrantCoversRoot(t *testing.T) {
	c, err := ParseSudoConfig([]byte(`users:
  all: {privileged: {tools: {}, resources: {"file://": [""]}}}
  slash: {privileged: {tools: {}, resources: {"file://": ["/"]}}}
  narrow: {privileged: {tools: {}, resources: {"file://": ["/var/log"]}}}
`), true)
	if err != nil {
		t.Fatal(err)
	}
	for user, want := range map[string]bool{"all": true, "slash": true, "narrow": false, "nobody": false} {
		if got := c.ResourceGrantCoversRoot(user, "file://"); got != want {
			t.Errorf("%s: got %t", user, got)
		}
	}
}

func TestPathsRequired(t *testing.T) {
	for _, tool := range []string{"files/read", "disks/usage", "disks/free"} {
		doc := "users:\n  a:\n    privileged:\n      tools:\n        " + tool + ": {allowed: true}\n"
		if _, err := ParseSudoConfig([]byte(doc), true); err == nil || !strings.Contains(err.Error(), "allowed without paths") {
			t.Errorf("%s allowed without paths: err = %v", tool, err)
		}
		if _, err := ParseSudoConfig([]byte(doc), false); err != nil {
			t.Errorf("%s: lenient (startup) must still load: %v", tool, err)
		}
	}
	ok := "users:\n  a:\n    privileged:\n      tools:\n        disks/usage: {allowed: true, paths: [\"/\"]}\n        files/read: {allowed: false}\n"
	if _, err := ParseSudoConfig([]byte(ok), true); err != nil {
		t.Errorf("explicit paths / not-allowed rejected: %v", err)
	}
}

func TestListeners(t *testing.T) {
	cases := []struct {
		doc  string
		want []Listener
		err  string
	}{
		// legacy: plain HTTP on server.port, TLS extra on tls.port
		{"server: {port: 9091}\n", []Listener{{9091, false}}, ""},
		{"users: []\n", []Listener{{9091, false}}, ""},
		{"server: {port: 9091, tls: {enabled: true}}\n", []Listener{{9443, true}, {9091, false}}, ""},
		// new layout: TLS on, plain HTTP off by default
		{"server: {tls: {enabled: true}, http: {enabled: false}}\n", []Listener{{9091, true}}, ""},
		{"server: {tls: {enabled: true}, http: {enabled: true}}\n", []Listener{{9091, true}, {9090, false}}, ""},
		{"server: {tls: {enabled: false}, http: {enabled: true, port: 8080}}\n", []Listener{{8080, false}}, ""},
		{"server: {tls: {enabled: false}, http: {enabled: false}}\n", nil, "no listener enabled"},
		{"server: {tls: {enabled: true, port: 9091}, http: {enabled: true, port: 9091}}\n", nil, "both set to port 9091"},
		{"server: {port: 9091, tls: {enabled: true}, http: {enabled: false}}\n", nil, "old plain-HTTP setting"},
	}
	for _, c := range cases {
		cfg, err := ParseDaemonConfig([]byte(c.doc), true)
		if c.err != "" {
			if err == nil || !strings.Contains(err.Error(), c.err) {
				t.Errorf("%q: want error %q, got %v", c.doc, c.err, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: %v", c.doc, err)
			continue
		}
		got := cfg.Listeners()
		if len(got) != len(c.want) {
			t.Errorf("%q: listeners %v, want %v", c.doc, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q: listeners %v, want %v", c.doc, got, c.want)
			}
		}
	}
	cfg, _ := ParseDaemonConfig([]byte("server: {tls: {enabled: true}, http: {enabled: false}}\n"), true)
	if cert, key := cfg.TLSFiles("/etc/mcpd/configs"); cert != "/etc/mcpd/configs/tls/mcpd.crt" || key != "/etc/mcpd/configs/tls/mcpd.key" {
		t.Errorf("default TLS files: %s %s", cert, key)
	}
}
