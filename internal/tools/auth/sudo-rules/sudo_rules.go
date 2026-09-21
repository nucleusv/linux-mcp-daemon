package sudorules

import (
	"encoding/json"
	"fmt"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
)

// GetSudoRulesArgs has no arguments since we just return the caller's rules.
type GetSudoRulesArgs struct {
	OutputFormat string `json:"output_format,omitempty"`
}

// GetSudoRules returns the subset of mcp-sudo.yaml rules applicable to the authenticated user.
func SudoRules(argsJSON []byte, username string, sudoConfig *config.SudoConfig) (string, error) {
	var args GetSudoRulesArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil && len(argsJSON) > 0 {
		// Ignore error if it's empty, since args are optional here
	}

	if sudoConfig == nil {
		if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
			return `{"error": "No sudo configuration loaded."}`, nil
		}
		return "No sudo configuration loaded.", nil
	}

	if userSudo, ok := sudoConfig.Users[username]; ok {
		bytes, err := json.MarshalIndent(userSudo.Privileged, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to encode rules: %v", err)
		}
		if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
			return string(bytes), nil
		}
		return fmt.Sprintf("Your authorized privileged tools:\n%s", string(bytes)), nil
	}

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		return `{}`, nil
	}
	return "You have no privileged tools authorized in mcp-sudo.yaml.", nil
}
