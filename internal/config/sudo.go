package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type SudoConfig struct {
	Rules []SudoRule `yaml:"rules"`
}

type SudoRule struct {
	User  string     `yaml:"user"`
	Tools []ToolRule `yaml:"tools"`
}

type ToolRule struct {
	Name  string `yaml:"name"`
	RunAs string `yaml:"run_as"`
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
	return &cfg, nil
}

// CanRunAsRoot checks if a specific user is authorized to run a specific tool as root.
func (c *SudoConfig) CanRunAsRoot(username, toolName string) bool {
	for _, rule := range c.Rules {
		if rule.User == username {
			for _, tool := range rule.Tools {
				if tool.Name == toolName && tool.RunAs == "root" {
					return true
				}
			}
		}
	}
	return false
}
