// Package containers lists Docker containers, like services/list does for
// systemd units.
package containers

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
)

type Args struct {
	docker.CommonArgs
	All          bool   `json:"all,omitempty"`
	Pattern      string `json:"pattern,omitempty"`
	State        string `json:"state,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

// apiContainer is the subset of GET /containers/json this tool reports.
type apiContainer struct {
	ID      string            `json:"Id"`
	Names   []string          `json:"Names"`
	Image   string            `json:"Image"`
	ImageID string            `json:"ImageID"`
	Command string            `json:"Command"`
	Created int64             `json:"Created"`
	State   string            `json:"State"`
	Status  string            `json:"Status"`
	Labels  map[string]string `json:"Labels"`
	Ports   []struct {
		IP          string `json:"IP"`
		PrivatePort int    `json:"PrivatePort"`
		PublicPort  int    `json:"PublicPort"`
		Type        string `json:"Type"`
	} `json:"Ports"`
}

func List(argsJSON []byte) (string, error) {
	var args Args
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	// No containers: check here - this tool names no container, and a
	// containers: list on it is a strict-mode config error (see sudo.go).
	c := args.Client(30 * time.Second)

	q := docker.Q("all", boolQ(args.All || args.State != ""), "limit", intQ(args.Limit))
	var list []apiContainer
	if err := c.GetJSON("/containers/json"+q, &list); err != nil {
		return "", err
	}

	rows := make([]map[string]interface{}, 0, len(list))
	var text strings.Builder
	for _, ct := range list {
		name := shortName(ct.Names)
		if args.Pattern != "" {
			if ok, _ := path.Match(args.Pattern, name); !ok {
				continue
			}
		}
		if args.State != "" && !strings.EqualFold(ct.State, args.State) {
			continue
		}
		ports := formatPorts(ct)
		if docker.Structured(args.OutputFormat) {
			rows = append(rows, map[string]interface{}{
				"name":     name,
				"names":    trimNames(ct.Names),
				"id":       short(ct.ID),
				"full_id":  ct.ID,
				"image":    ct.Image,
				"image_id": ct.ImageID,
				"command":  ct.Command,
				"state":    ct.State,
				"status":   ct.Status,
				"created":  time.Unix(ct.Created, 0).UTC().Format(time.RFC3339),
				"ports":    ports,
				"labels":   ct.Labels,
			})
			continue
		}
		fmt.Fprintf(&text, "[%s] %s (%s)\n  Image: %s\n  Status: %s\n", ct.State, name, short(ct.ID), ct.Image, ct.Status)
		if ports != "" {
			fmt.Fprintf(&text, "  Ports: %s\n", ports)
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
		return "No containers found matching the criteria.", nil
	}
	text.WriteString("Hint: For one container's runtime summary, read docker-container://<name>/status; for the full config, docker-container://<name>/inspect.\n")
	return text.String(), nil
}

// shortName is the primary name Docker shows in `docker ps`, without the
// leading slash the API adds.
func shortName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.TrimPrefix(names[0], "/")
}

func trimNames(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, strings.TrimPrefix(n, "/"))
	}
	return out
}

func short(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// formatPorts renders published and exposed ports the way `docker ps` does,
// e.g. "0.0.0.0:8080->80/tcp, 9090/tcp".
func formatPorts(ct apiContainer) string {
	seen := map[string]bool{}
	var parts []string
	for _, p := range ct.Ports {
		var s string
		if p.PublicPort != 0 {
			host := p.IP
			if host == "" {
				host = "0.0.0.0"
			}
			s = fmt.Sprintf("%s:%d->%d/%s", host, p.PublicPort, p.PrivatePort, p.Type)
		} else {
			s = fmt.Sprintf("%d/%s", p.PrivatePort, p.Type)
		}
		if !seen[s] {
			seen[s] = true
			parts = append(parts, s)
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func boolQ(b bool) string {
	if b {
		return "true"
	}
	return ""
}

func intQ(n int) string {
	if n > 0 {
		return strconv.Itoa(n)
	}
	return ""
}
