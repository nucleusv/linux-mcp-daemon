// Package logs reads a container's log stream, in the shape of
// logs/journal-control.
package logs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
)

// maxLogBytes caps one answer. A container that has logged a gigabyte must
// not turn into a gigabyte of response; ask for a smaller tail instead.
const maxLogBytes = 1 << 20

type Args struct {
	docker.CommonArgs
	Container    string `json:"container,omitempty"`
	Lines        int    `json:"lines,omitempty"`
	Since        string `json:"since,omitempty"`
	Until        string `json:"until,omitempty"`
	Timestamps   bool   `json:"timestamps,omitempty"`
	Stdout       *bool  `json:"stdout,omitempty"`
	Stderr       *bool  `json:"stderr,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

func Logs(argsJSON []byte) (string, error) {
	var args Args
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	c := args.Client(60 * time.Second)
	id, name, err := docker.Authorize(c, args.Container, args.Containers)
	if err != nil {
		return "", err
	}

	tail := 100
	if args.Lines > 0 {
		tail = args.Lines
	}
	since, err := engineTime(args.Since)
	if err != nil {
		return "", err
	}
	until, err := engineTime(args.Until)
	if err != nil {
		return "", err
	}
	q := docker.Q(
		"stdout", onByDefault(args.Stdout),
		"stderr", onByDefault(args.Stderr),
		"tail", strconv.Itoa(tail),
		"since", since,
		"until", until,
		"timestamps", boolQ(args.Timestamps),
	)
	body, err := c.GetStream("/containers/" + id + "/logs" + q)
	if err != nil {
		return "", err
	}
	defer body.Close()

	// Combined, not split: `docker logs` shows both streams interleaved in
	// the order they were recorded, and splitting them loses the chronology.
	out, err := docker.DemuxCombined(body, maxLogBytes)
	if err != nil {
		return "", fmt.Errorf("reading logs of %s: %v", name, err)
	}

	if docker.Structured(args.OutputFormat) {
		b, _ := json.Marshal(map[string]interface{}{
			"container": name,
			"id":        id,
			"lines":     tail,
			"logs":      out,
		})
		return string(b), nil
	}
	if out == "" {
		return fmt.Sprintf("Container %s (%s) has no log output for this selection.\n", name, id[:12]), nil
	}
	return out, nil
}

// engineTime converts a time for the Engine API's since/until, which take only
// a unix timestamp - `docker logs --since 2026-09-27T04:28:30Z` works because
// the CLI parses it first, and this tool is that CLI's replacement.
func engineTime(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		return v, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return "", fmt.Errorf("invalid time %q: give a unix timestamp (1759005000) or an RFC3339 time (2026-09-27T04:28:30Z)", v)
	}
	return strconv.FormatInt(t.Unix(), 10), nil
}

// onByDefault: both streams are on unless the caller explicitly says false.
func onByDefault(b *bool) string {
	if b != nil && !*b {
		return ""
	}
	return "true"
}

func boolQ(b bool) string {
	if b {
		return "true"
	}
	return ""
}
