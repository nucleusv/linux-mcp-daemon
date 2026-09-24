package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/netpolicy"
)

type SudoConfig struct {
	Users map[string]UserSudo `yaml:"users"`
}

type UserSudo struct {
	Privileged PrivilegedConfig `yaml:"privileged"`
}

type PrivilegedConfig struct {
	Tools     map[string]ToolPrivilege `yaml:"tools"`
	Resources map[string][]string      `yaml:"resources,omitempty"`
}

type ToolPrivilege struct {
	Allowed bool     `yaml:"allowed"`
	Paths   []string `yaml:"paths"`
	// Network restricts where an outbound network tool (network/curl,
	// network/ping) may connect. Unlike Allowed, which only governs
	// running as root, it applies to every call of the tool - network
	// access doesn't depend on the worker's uid. Absent = unrestricted.
	Network *netpolicy.Policy `yaml:"network,omitempty"`
	// Sysctl restricts kernel/system-control writes. Absent = unrestricted
	// (writes allowed wherever the tool may run as root, as before).
	Sysctl *SysctlPolicy `yaml:"sysctl,omitempty"`
}

// SysctlPolicy limits which kernel parameters a user may change. Writing
// some parameters is equivalent to running code as root (e.g.
// kernel.core_pattern, kernel.modprobe), so it's worth being able to allow
// writes to only a known set of keys. (For read-only access, don't grant
// `allowed` at all: reading needs no root, and without root the OS refuses
// every write.)
type SysctlPolicy struct {
	// RemovedReadOnly catches the former read_only option. It was removed
	// as redundant with simply not granting `allowed`, and is kept only so
	// a leftover `read_only: true` fails loudly at load - a silently ignored
	// key would leave writes open while the config reads as closed.
	RemovedReadOnly *bool `yaml:"read_only"`
	// WriteKeys, if non-empty, is the only set of keys that may be
	// written. Entries are dotted keys or glob patterns ("vm.*",
	// "net.ipv4.conf.*.rp_filter"); "*" matches within one dotted
	// component.
	WriteKeys []string `yaml:"write_keys,omitempty"`
}

// CanWriteSysctl reports whether username may write key (dotted form)
// through kernel/system-control, and why not if they can't.
func (c *SudoConfig) CanWriteSysctl(username, key string) (bool, string) {
	userSudo, ok := c.Users[username]
	if !ok {
		return true, ""
	}
	privs, ok := userSudo.Privileged.Tools["kernel/system-control"]
	if !ok || privs.Sysctl == nil {
		return true, ""
	}
	if len(privs.Sysctl.WriteKeys) == 0 {
		return true, ""
	}
	for _, pattern := range privs.Sysctl.WriteKeys {
		if sysctlKeyMatch(pattern, key) {
			return true, ""
		}
	}
	return false, fmt.Sprintf("writing %s is not permitted for this user (not in sysctl.write_keys in mcp-sudo.yaml)", key)
}

func validSysctlPattern(pattern string) error {
	_, err := path.Match(strings.ReplaceAll(pattern, ".", "/"), "")
	return err
}

// sysctlKeyMatch matches a dotted key against a dotted glob, treating "."
// as the separator so "*" never spans components ("vm.*" matches
// "vm.swappiness" but not "vm.a.b").
func sysctlKeyMatch(pattern, key string) bool {
	ok, err := path.Match(strings.ReplaceAll(pattern, ".", "/"), strings.ReplaceAll(key, ".", "/"))
	return err == nil && ok
}

func LoadSudoConfig(path string) (*SudoConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := ParseSudoConfig(data, false)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// LoadSudoConfigStrict is LoadSudoConfig rejecting unknown keys too: a
// misspelled key (`path:` for `paths:`) is an error rather than a rule
// silently left out. Used by daemon/reload-config and linuxctl edit.
func LoadSudoConfigStrict(path string) (*SudoConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := ParseSudoConfig(data, true)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// ParseSudoConfig parses and validates mcp-sudo.yaml; strict rejects
// unknown keys.
func ParseSudoConfig(data []byte, strict bool) (*SudoConfig, error) {
	var cfg SudoConfig
	if err := decodeYAML(data, &cfg, strict); err != nil {
		return nil, err
	}

	// Validation
	for username, userSudo := range cfg.Users {
		for toolName, privs := range userSudo.Privileged.Tools {
			if privs.Sysctl != nil {
				if privs.Sysctl.RemovedReadOnly != nil {
					return nil, fmt.Errorf("user '%s', tool '%s': sysctl.read_only is no longer supported - for read-only access remove `allowed` (reading needs no root, and without root the OS refuses writes); to limit writes use sysctl.write_keys", username, toolName)
				}
				for _, p := range privs.Sysctl.WriteKeys {
					if err := validSysctlPattern(p); err != nil {
						return nil, fmt.Errorf("user '%s', tool '%s': invalid sysctl.write_keys pattern %q: %v", username, toolName, p, err)
					}
				}
			}
			if privs.Network != nil {
				if err := privs.Network.Validate(); err != nil {
					return nil, fmt.Errorf("user '%s', tool '%s': %v", username, toolName, err)
				}
			}
			if _, takesPath := PathTools[toolName]; len(privs.Paths) > 0 && !takesPath && strict {
				return nil, fmt.Errorf("user '%s', tool '%s': paths has no effect - %s takes no path argument (paths can limit: %s)", username, toolName, toolName, strings.Join(sortedPathTools(), ", "))
			}
		}
	}

	return &cfg, nil
}

// CanRunAsRoot checks if a specific user is authorized to run a specific tool as root.
func (c *SudoConfig) CanRunAsRoot(username, toolName string) bool {
	if userSudo, ok := c.Users[username]; ok {
		if privs, ok := userSudo.Privileged.Tools[toolName]; ok {
			return privs.Allowed
		}
	}
	return false
}

// NetworkPolicy returns the network restrictions configured for this
// user's use of toolName, or nil for none.
func (c *SudoConfig) NetworkPolicy(username, toolName string) *netpolicy.Policy {
	if userSudo, ok := c.Users[username]; ok {
		if privs, ok := userSudo.Privileged.Tools[toolName]; ok {
			return privs.Network
		}
	}
	return nil
}

// GetAllowedPaths fetches the restricted paths for a tool.
// PathTools are the tools that take a filesystem `path` argument, so a
// grant for them can carry `paths:`. The value says whether running them as
// root requires `paths:` (files/*: no paths, no root) or merely honors it
// when present (absent = any path). `paths:` on any other tool would be a
// restriction that restricts nothing, so the config loader rejects it.
var PathTools = map[string]bool{
	"files/list":     true,
	"files/read":     true,
	"files/create":   true,
	"files/update":   true,
	"files/find":     true,
	"files/filetype": true,
	"files/chmod":    true,
	"files/chown":    true,
	"disks/free":     false,
	"disks/usage":    false,
}

func (c *SudoConfig) GetAllowedPaths(username, toolName string) []string {
	if userSudo, ok := c.Users[username]; ok {
		if privs, ok := userSudo.Privileged.Tools[toolName]; ok {
			return privs.Paths
		}
	}
	return nil
}

// PathAllowed reports whether path lies inside one of the allowed
// directories, returning the cleaned path the caller must then operate on.
// The check is lexical and boundary-aware: path is cleaned first (so
// "/tmp/../etc/shadow" is judged as "/etc/shadow", not waved through by a
// raw prefix match on "/tmp"), and an allowed "/tmp" covers "/tmp" and
// "/tmp/x" but not "/tmpfoo". Relative paths are always rejected - they'd be
// resolved against the worker's working directory, which no allowlist entry
// describes.
func PathAllowed(path string, allowed []string) (string, bool) {
	if !strings.HasPrefix(path, "/") {
		return "", false
	}
	clean := filepath.Clean(path)
	for _, a := range allowed {
		if !strings.HasPrefix(a, "/") {
			continue
		}
		a = filepath.Clean(a)
		if a == "/" || clean == a || strings.HasPrefix(clean, a+"/") {
			return clean, true
		}
	}
	return "", false
}

// CanReadResourceAsRoot checks if a specific user is authorized to read a resource path as root.
func (c *SudoConfig) CanReadResourceAsRoot(username, scheme, resourcePath string) bool {
	if userSudo, ok := c.Users[username]; ok {
		if resPaths, ok := userSudo.Privileged.Resources[scheme]; ok {
			for _, p := range resPaths {
				switch {
				// Exact match - how exact-match resources (devices://usb,
				// os://uname, ...) are granted: callers pass the literal
				// sentinel "*" and the config lists "*".
				case resourcePath == p:
					return true
				// Empty prefix grants a whole prefix-matched scheme (file://,
				// service://, process://). A literal "*" deliberately does
				// NOT - see the comment in configs/mcp-sudo.yaml.
				case p == "":
					return true
				// Filesystem path grants are matched on the cleaned path with
				// a directory boundary (see PathAllowed), so neither
				// "/var/log/../../etc/shadow" nor "/var/logs-private" gets
				// through a "/var/log" grant.
				case strings.HasPrefix(p, "/"):
					if _, ok := PathAllowed(resourcePath, []string{p}); ok {
						return true
					}
				// Non-path prefixes (service names, PIDs) keep plain prefix
				// matching, as before.
				case !strings.HasPrefix(resourcePath, "/") && strings.HasPrefix(resourcePath, p):
					return true
				}
			}
		}
	}
	return false
}

func sortedPathTools() []string {
	names := make([]string, 0, len(PathTools))
	for n := range PathTools {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
