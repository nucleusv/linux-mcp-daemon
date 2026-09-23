package dmesg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
)

// DmesgArgs defines the parameters for the logs/dmesg tool.
type DmesgArgs struct {
	Level        string `json:"level,omitempty"`         // Level filters by log level (e.g., "err,warn").
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format.
}

var validLevels = regexp.MustCompile(`^(emerg|alert|crit|err|warn|notice|info|debug)(,(emerg|alert|crit|err|warn|notice|info|debug))*$`)

func Dmesg(argsJSON []byte) (string, error) {
	var args DmesgArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	cmdArgs := []string{"--human"}

	if args.Level != "" {
		if !validLevels.MatchString(args.Level) {
			return "", fmt.Errorf("invalid level %q: expected comma-separated emerg,alert,crit,err,warn,notice,info,debug", args.Level)
		}
		cmdArgs = append(cmdArgs, "--level", args.Level)
	}

	// Wait, some busybox dmesg might not support these flags, but standard util-linux does.
	// Let's rely on standard dmesg.

	cmd := exec.Command("dmesg", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("dmesg error: %v, stderr: %s", err, stderr.String())
	}

	// Truncation: dmesg can be huge. The prompt for Context is important.
	// Actually, the AI receives everything if we don't truncate, so let's keep it under a safe limit or let the user pipe it.
	// We will cap it at 30KB just in case.
	outStr := stdout.String()
	if len(outStr) > 30720 {
		outStr = outStr[len(outStr)-30720:]
		outStr = "[WARNING: Output truncated to last 30KB]\n...\n" + outStr
	}

	return outStr, nil
}
