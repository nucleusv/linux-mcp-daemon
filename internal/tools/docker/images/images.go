// Package images lists Docker images. Read-only: no pull, no build, no
// remove.
package images

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
)

type Args struct {
	docker.CommonArgs
	All          bool   `json:"all,omitempty"`
	Pattern      string `json:"pattern,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

type apiImage struct {
	ID          string            `json:"Id"`
	RepoTags    []string          `json:"RepoTags"`
	RepoDigests []string          `json:"RepoDigests"`
	Created     int64             `json:"Created"`
	Size        int64             `json:"Size"`
	Containers  int64             `json:"Containers"`
	Labels      map[string]string `json:"Labels"`
}

func List(argsJSON []byte) (string, error) {
	var args Args
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	c := args.Client(30 * time.Second)
	q := ""
	if args.All {
		q = docker.Q("all", "true")
	}
	var list []apiImage
	if err := c.GetJSON("/images/json"+q, &list); err != nil {
		return "", err
	}

	rows := make([]map[string]interface{}, 0, len(list))
	var text strings.Builder
	for _, img := range list {
		tags := img.RepoTags
		if len(tags) == 0 {
			tags = []string{"<none>:<none>"}
		}
		if args.Pattern != "" && !matchAny(tags, args.Pattern) {
			continue
		}
		if docker.Structured(args.OutputFormat) {
			rows = append(rows, map[string]interface{}{
				"id":           short(img.ID),
				"full_id":      img.ID,
				"repo_tags":    tags,
				"repo_digests": img.RepoDigests,
				"size_bytes":   img.Size,
				"created":      time.Unix(img.Created, 0).UTC().Format(time.RFC3339),
				"containers":   img.Containers,
				"labels":       img.Labels,
			})
			continue
		}
		fmt.Fprintf(&text, "%s\n  ID: %s | Size: %s | Created: %s\n\n",
			strings.Join(tags, ", "), short(img.ID), humanSize(img.Size),
			time.Unix(img.Created, 0).UTC().Format("2006-01-02 15:04:05 UTC"))
	}

	if docker.Structured(args.OutputFormat) {
		if len(rows) == 0 {
			return "[]", nil
		}
		b, _ := json.Marshal(rows)
		return string(b), nil
	}
	if text.Len() == 0 {
		return "No images found matching the criteria.", nil
	}
	text.WriteString("Hint: For one image's layers, env and entrypoint, read image://<name>/inspect\n")
	return text.String(), nil
}

func matchAny(tags []string, pattern string) bool {
	for _, t := range tags {
		if ok, _ := path.Match(pattern, t); ok {
			return true
		}
		// A pattern without a tag should still match "nginx:1.25".
		if repo := strings.SplitN(t, ":", 2)[0]; repo != t {
			if ok, _ := path.Match(pattern, repo); ok {
				return true
			}
		}
	}
	return false
}

// short trims Docker's "sha256:" prefix and keeps the usual 12 characters.
func short(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func humanSize(n int64) string {
	const unit = 1024.0
	v := float64(n)
	for _, suffix := range []string{"B", "KiB", "MiB", "GiB", "TiB"} {
		if v < unit || suffix == "TiB" {
			return fmt.Sprintf("%.1f %s", v, suffix)
		}
		v /= unit
	}
	return fmt.Sprintf("%d B", n)
}
