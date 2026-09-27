// Package volumes lists Docker volumes. Read-only: no create, no remove.
package volumes

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
)

type Args struct {
	docker.CommonArgs
	Pattern      string `json:"pattern,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

type apiVolume struct {
	Name       string            `json:"Name"`
	Driver     string            `json:"Driver"`
	Mountpoint string            `json:"Mountpoint"`
	CreatedAt  string            `json:"CreatedAt"`
	Scope      string            `json:"Scope"`
	Labels     map[string]string `json:"Labels"`
	Options    map[string]string `json:"Options"`
}

func List(argsJSON []byte) (string, error) {
	var args Args
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	c := args.Client(30 * time.Second)
	var answer struct {
		Volumes  []apiVolume `json:"Volumes"`
		Warnings []string    `json:"Warnings"`
	}
	if err := c.GetJSON("/volumes", &answer); err != nil {
		return "", err
	}
	users := volumeUsers(c)

	rows := make([]map[string]interface{}, 0, len(answer.Volumes))
	var text strings.Builder
	for _, v := range answer.Volumes {
		if args.Pattern != "" {
			if ok, _ := path.Match(args.Pattern, v.Name); !ok {
				continue
			}
		}
		if docker.Structured(args.OutputFormat) {
			rows = append(rows, map[string]interface{}{
				"name":       v.Name,
				"driver":     v.Driver,
				"mountpoint": v.Mountpoint,
				"created":    v.CreatedAt,
				"scope":      v.Scope,
				"labels":     v.Labels,
				"options":    v.Options,
				"in_use_by":  users[v.Name],
			})
			continue
		}
		fmt.Fprintf(&text, "%s\n  Driver: %s | Scope: %s\n  Mountpoint: %s\n", v.Name, v.Driver, v.Scope, v.Mountpoint)
		if in := users[v.Name]; len(in) > 0 {
			fmt.Fprintf(&text, "  In use by: %s\n", strings.Join(in, ", "))
		} else {
			text.WriteString("  In use by: (nothing)\n")
		}
		text.WriteString("\n")
	}

	if docker.Structured(args.OutputFormat) {
		if len(rows) == 0 {
			return "[]", nil
		}
		b, _ := json.Marshal(rows)
		return string(b), nil
	}
	if text.Len() == 0 {
		return "No volumes found matching the criteria.", nil
	}
	for _, w := range answer.Warnings {
		fmt.Fprintf(&text, "Warning: %s\n", w)
	}
	text.WriteString("Hint: For one volume's options and labels, read volume://<name>/inspect\n")
	return text.String(), nil
}

// volumeUsers maps each named volume to the containers mounting it.
//
// The Engine API reports RefCount only via /system/df, and a count is less
// useful than the names, so this walks the container list instead. A failure
// here is not fatal: the listing is still worth returning without it.
func volumeUsers(c *docker.Client) map[string][]string {
	var list []struct {
		Names  []string `json:"Names"`
		Mounts []struct {
			Type string `json:"Type"`
			Name string `json:"Name"`
		} `json:"Mounts"`
	}
	if err := c.GetJSON("/containers/json"+docker.Q("all", "true"), &list); err != nil {
		return nil
	}
	users := map[string][]string{}
	for _, ct := range list {
		name := ""
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		for _, m := range ct.Mounts {
			if m.Type == "volume" && m.Name != "" {
				users[m.Name] = append(users[m.Name], name)
			}
		}
	}
	for k := range users {
		sort.Strings(users[k])
	}
	return users
}
