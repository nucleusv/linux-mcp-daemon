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
