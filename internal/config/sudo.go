package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
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
}

func LoadSudoConfig(path string) (*SudoConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg SudoConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Validation
	for username, userSudo := range cfg.Users {
		for toolName, privs := range userSudo.Privileged.Tools {
			if toolName == "list_files" && privs.Allowed {
				if len(privs.Paths) == 0 {
					return nil, fmt.Errorf("validation error in %s: user '%s' has list_files allowed but no paths specified. 'paths' array must not be empty", path, username)
				}
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

// GetAllowedPaths fetches the restricted paths for a tool.
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
