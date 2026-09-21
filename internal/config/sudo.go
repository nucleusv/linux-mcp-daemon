package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type SudoConfig struct {
	Users map[string]UserSudo `yaml:"users"`
}

type UserSudo struct {
	Privileged map[string]ToolPrivilege `yaml:"privileged"`
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
		for toolName, privs := range userSudo.Privileged {
			if toolName == "list_directory" && privs.Allowed {
				if len(privs.Paths) == 0 {
					return nil, fmt.Errorf("validation error in %s: user '%s' has list_directory allowed but no paths specified. 'paths' array must not be empty", path, username)
				}
			}
		}
	}

	return &cfg, nil
}

// CanRunAsRoot checks if a specific user is authorized to run a specific tool as root.
func (c *SudoConfig) CanRunAsRoot(username, toolName string) bool {
	if userSudo, ok := c.Users[username]; ok {
		if privs, ok := userSudo.Privileged[toolName]; ok {
			return privs.Allowed
		}
	}
	return false
}

// GetAllowedPaths fetches the restricted paths for a tool.
func (c *SudoConfig) GetAllowedPaths(username, toolName string) []string {
	if userSudo, ok := c.Users[username]; ok {
		if privs, ok := userSudo.Privileged[toolName]; ok {
			return privs.Paths
		}
	}
	return nil
}
