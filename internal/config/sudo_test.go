package config

import (
	"os"
	"testing"
)

func TestPathAllowed(t *testing.T) {
	cases := []struct {
		path    string
		allowed []string
		ok      bool
		clean   string
	}{
		{"/tmp/x", []string{"/tmp"}, true, "/tmp/x"},
		{"/tmp", []string{"/tmp"}, true, "/tmp"},
		{"/tmp/", []string{"/tmp/"}, true, "/tmp"},
		{"/tmp/../etc/shadow", []string{"/tmp"}, false, ""}, // climbing out via ..
		{"/tmpfoo/secret", []string{"/tmp"}, false, ""},     // prefix without boundary
		{"/tmp/a/../b", []string{"/tmp"}, true, "/tmp/b"},
		{"tmp/x", []string{"/tmp"}, false, ""}, // relative
		{"", []string{"/"}, false, ""},
		{"/etc/shadow", []string{"/"}, true, "/etc/shadow"},
		{"/../../etc", []string{"/"}, true, "/etc"},
		{"/var/log/syslog", nil, false, ""},
		{"/var/log/syslog", []string{"relative"}, false, ""},
	}
	for _, c := range cases {
		clean, ok := PathAllowed(c.path, c.allowed)
		if ok != c.ok || clean != c.clean {
			t.Errorf("PathAllowed(%q, %v) = %q, %v; want %q, %v", c.path, c.allowed, clean, ok, c.clean, c.ok)
		}
	}
}

func TestCanReadResourceAsRoot(t *testing.T) {
	c := &SudoConfig{Users: map[string]UserSudo{
		"u": {Privileged: PrivilegedConfig{Resources: map[string][]string{
			"file://":    {"/var/log"},
			"process://": {""},
			"os://uname": {"*"},
			"service://": {"*", "ssh"},
		}}},
	}}
	cases := []struct {
		scheme, path string
		want         bool
	}{
		{"file://", "/var/log/syslog", true},
		{"file://", "/var/log/../../etc/shadow", false},
		{"file://", "/var/logs-private/x", false},
		{"process://", "123", true},
		{"os://uname", "*", true},
		{"devices://pci", "*", false},
		// "*" is a literal on prefix-matched schemes, never "grant all".
		{"service://", "cron.service", false},
		{"service://", "ssh.service", true}, // plain prefix, as before
		{"file://", "*", false},
	}
	for _, tc := range cases {
		if got := c.CanReadResourceAsRoot("u", tc.scheme, tc.path); got != tc.want {
			t.Errorf("CanReadResourceAsRoot(%q, %q) = %v, want %v", tc.scheme, tc.path, got, tc.want)
		}
	}
}

func TestLoadPerToolNetworkPolicy(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/mcp-sudo.yaml"
	os.WriteFile(path, []byte(`users:
  alice:
    privileged:
      tools:
        network/curl:
          allowed: true
          network:
            deny_private: true
            allow: ["10.0.5.0/24"]
        network/ping:
          allowed: true
`), 0o600)
	cfg, err := LoadSudoConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	pol := cfg.NetworkPolicy("alice", "network/curl")
	if pol == nil || !pol.DenyPrivate || len(pol.Allow) != 1 {
		t.Errorf("curl policy not loaded: %+v", pol)
	}
	if cfg.NetworkPolicy("alice", "network/ping") != nil {
		t.Error("ping has no network block - must stay unrestricted")
	}
	if cfg.NetworkPolicy("bob", "network/curl") != nil {
		t.Error("unknown user must be unrestricted")
	}

	os.WriteFile(path, []byte(`users:
  alice:
    privileged:
      tools:
        network/curl:
          allowed: true
          network:
            allow: ["10.0.0.0/99"]
`), 0o600)
	if _, err := LoadSudoConfig(path); err == nil {
		t.Error("invalid CIDR accepted at load time")
	}
}
