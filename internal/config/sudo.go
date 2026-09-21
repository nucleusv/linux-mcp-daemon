package config

import (
	"fmt"
	"os"
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
			if toolName == "get_list_of_files" && privs.Allowed {
				if len(privs.Paths) == 0 {
					return nil, fmt.Errorf("validation error in %s: user '%s' has get_list_of_files allowed but no paths specified. 'paths' array must not be empty", path, username)
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

// CanReadResourceAsRoot checks if a specific user is authorized to read a resource path as root.
func (c *SudoConfig) CanReadResourceAsRoot(username, scheme, resourcePath string) bool {
	if userSudo, ok := c.Users[username]; ok {
		if resPaths, ok := userSudo.Privileged.Resources[scheme]; ok {
			for _, p := range resPaths {
				// Exact match or prefix match for directories
				if resourcePath == p || strings.HasPrefix(resourcePath, p) {
					return true
				}
			}
		}
	}
	return false
}
