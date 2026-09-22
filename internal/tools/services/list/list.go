package list

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"
)

type GetListArgs struct {
	Pattern      string `json:"pattern,omitempty"`
	ActiveState  string `json:"active_state,omitempty"`
	LoadState    string `json:"load_state,omitempty"`
	SubState     string `json:"sub_state,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

func List(argsJSON []byte) (string, error) {
	var args GetListArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	conn, err := dbus.NewSystemdConnectionContext(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to connect to systemd dbus: %v", err)
	}
	defer conn.Close()

	units, err := conn.ListUnitsContext(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to list units: %v", err)
	}

	var jsonResult []map[string]interface{}
	var textResult string

	// Basic wildcard to Go strings.Contains or HasPrefix/Suffix approach for simple pattern matching
	matchPattern := func(name string, pattern string) bool {
		if pattern == "" {
			return true
		}
		if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
			return strings.Contains(name, strings.Trim(pattern, "*"))
		} else if strings.HasPrefix(pattern, "*") {
			return strings.HasSuffix(name, strings.TrimPrefix(pattern, "*"))
		} else if strings.HasSuffix(pattern, "*") {
			return strings.HasPrefix(name, strings.TrimSuffix(pattern, "*"))
		}
		return name == pattern
	}

	for _, u := range units {
		// Only show services
		if !strings.HasSuffix(u.Name, ".service") {
			continue
		}

		// Apply filters
		if args.ActiveState != "" && u.ActiveState != args.ActiveState {
			continue
		}
		if args.LoadState != "" && u.LoadState != args.LoadState {
			continue
		}
		if args.SubState != "" && u.SubState != args.SubState {
			continue
		}
		if !matchPattern(u.Name, args.Pattern) {
			continue
		}

		if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
			jsonResult = append(jsonResult, map[string]interface{}{
				"name":         u.Name,
				"description":  u.Description,
				"load_state":   u.LoadState,
				"active_state": u.ActiveState,
				"sub_state":    u.SubState,
			})
		} else {
			textResult += fmt.Sprintf("[%s] %s\n  State: %s (%s) | Load: %s\n  Desc: %s\n\n",
				u.ActiveState, u.Name, u.SubState, u.ActiveState, u.LoadState, u.Description)
		}
	}

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		if len(jsonResult) == 0 {
			return "[]", nil
		}
		b, _ := json.Marshal(jsonResult)
		return string(b), nil
	}

	if textResult == "" {
		return "No services found matching the criteria.", nil
	}

	textResult += "Hint: To read detailed properties of a specific service, use the resource: service://<name>/status\n"

	return textResult, nil
}
