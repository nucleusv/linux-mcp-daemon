// Package manage performs container lifecycle actions, like services/manage
// does for systemd units.
//
// Unlike services/manage, this tool has one irreversible verb - remove - so
// that one is fenced: it is the only non-POST action, and it sends neither
// Docker's `force` nor its `v`. Removing a running container is kill then
// remove, two separate audited calls; anonymous volumes are never deleted
// here (that is FR-012's prune, behind its own allowlist).
package manage

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
)

type Args struct {
	docker.CommonArgs
	Container    string `json:"container,omitempty"`
	Action       string `json:"action,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

// actions are the POST lifecycle endpoints; remove is handled separately
// because it is a DELETE and because it is the destructive one.
var actions = map[string]string{
	"start":   "start",
	"stop":    "stop",
	"restart": "restart",
	"kill":    "kill",
	"pause":   "pause",
	"unpause": "unpause",
}

func Manage(argsJSON []byte) (string, error) {
	var args Args
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}
	action := strings.ToLower(strings.TrimSpace(args.Action))
	endpoint, known := actions[action]
	if !known && action != "remove" {
		return "", fmt.Errorf("unknown action %q: use one of %s, remove", args.Action, strings.Join(sortedActions(), ", "))
	}

	c := args.Client(60 * time.Second)
	id, name, err := docker.Authorize(c, args.Container, args.Containers)
	if err != nil {
		return "", err
	}

	// Act on the resolved ID, never on the caller's string: the ID is what
	// was authorized, and a concurrent `docker rename` cannot re-point it.
	if action == "remove" {
		if err := c.Delete("/containers/" + id); err != nil {
			var apiErr *docker.APIError
			if errors.As(err, &apiErr) && apiErr.Status == 409 {
				return "", fmt.Errorf("%v - stop or kill it first (this tool never removes a running container by force)", err)
			}
			return "", err
		}
	}
	result := "ok"
	if action != "remove" {
		if err := c.PostJSON("/containers/"+id+"/"+endpoint, nil, nil); err != nil {
			// Docker answers 304 when the container is already in the
			// requested state (start on a running one, stop on a stopped
			// one). That is a no-op, not a failure.
			var apiErr *docker.APIError
			if !errors.As(err, &apiErr) || apiErr.Status != 304 {
				return "", err
			}
			result = "unchanged"
		}
	}

	if docker.Structured(args.OutputFormat) {
		b, _ := json.Marshal(map[string]interface{}{
			"container": name,
			"id":        id,
			"action":    action,
			"result":    result,
		})
		return string(b), nil
	}
	if result == "unchanged" {
		return fmt.Sprintf("Container %s (%s): %s not needed, it is already in that state.\n", name, id[:12], action), nil
	}
	return fmt.Sprintf("Container %s (%s): %s succeeded.\n", name, id[:12], action), nil
}

func sortedActions() []string {
	out := make([]string, 0, len(actions))
	for a := range actions {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}
