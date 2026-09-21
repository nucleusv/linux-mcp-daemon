package tools

import (
	"encoding/json"
	"fmt"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
)

// GetSudoRulesArgs has no arguments since we just return the caller's rules.
type GetSudoRulesArgs struct{}

// GetSudoRules returns the subset of mcp-sudo.yaml rules applicable to the authenticated user.
func GetSudoRules(username string, sudoCfg *config.SudoConfig) (string, error) {
	if sudoCfg == nil {
		return "No sudo configuration loaded.", nil
	}

	for _, rule := range sudoCfg.Rules {
		if rule.User == username {
			bytes, err := json.MarshalIndent(rule.Tools, "", "  ")
			if err != nil {
				return "", fmt.Errorf("failed to encode rules: %v", err)
			}
			return fmt.Sprintf("Your authorized privileged tools:\n%s", string(bytes)), nil
		}
	}

	return "You have no privileged tools authorized in mcp-sudo.yaml.", nil
}
