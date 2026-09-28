// Package prune reclaims unused Docker objects - the one docker/* tool that
// deletes things it was never told the names of.
//
// Every other destructive docker tool is scoped by container name
// (docker/manage's containers: list). Prune cannot be: it is scoped by what
// kind of garbage to collect, so it carries its own allowlist (`prune:` in
// mcp-sudo.yaml) and the requested target is checked against it before the
// socket is dialled at all.
//
// One target per call, deliberately. Kinds are not independent - pruning
// containers makes their images dangling and their anonymous volumes unused -
// so a multi-target call quietly reclaims more than the sum of its parts, and
// the caller cannot tell from the request which of those deletions it asked
// for. Separate calls make each deletion its own decision, and its own audit
// line.
//
// Two Engine API defaults are load-bearing here and are deliberately never
// overridden: /images/prune removes only *dangling* images (sending
// dangling=false would remove every image no container currently uses), and
// /volumes/prune removes only *anonymous* volumes (all=true would take named
// ones, which hold the data nothing can rebuild). This tool sends no filters
// at all, so both defaults stand.
package prune

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
)

type Args struct {
	docker.CommonArgs
	// Prune is this user's prune: allowlist, injected by the daemon like
	// CommonArgs.Containers. Empty refuses everything.
	Prune        []string `json:"_prune,omitempty"`
	Target       string   `json:"target,omitempty"`
	OutputFormat string   `json:"output_format,omitempty"`
}

var endpoints = map[string]string{
	"containers":  "/containers/prune",
	"images":      "/images/prune",
	"volumes":     "/volumes/prune",
	"networks":    "/networks/prune",
	"build-cache": "/build/prune",
}

// apiResult is every prune response in one struct: each endpoint fills the
// one list that is its own and leaves the rest absent. /networks/prune is the
// only one that reports no SpaceReclaimed.
type apiResult struct {
	ContainersDeleted []string `json:"ContainersDeleted"`
	VolumesDeleted    []string `json:"VolumesDeleted"`
	NetworksDeleted   []string `json:"NetworksDeleted"`
	CachesDeleted     []string `json:"CachesDeleted"`
	ImagesDeleted     []struct {
		Untagged string `json:"Untagged"`
		Deleted  string `json:"Deleted"`
	} `json:"ImagesDeleted"`
	SpaceReclaimed int64 `json:"SpaceReclaimed"`
}

type targetResult struct {
	Target    string   `json:"target"`
	Deleted   []string `json:"deleted"`
	Count     int      `json:"count"`
	Reclaimed int64    `json:"space_reclaimed_bytes"`
}

func Prune(argsJSON []byte) (string, error) {
	var args Args
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}
	target := strings.ToLower(strings.TrimSpace(args.Target))
	if target == "" {
		return "", fmt.Errorf("target is required: name the one kind to reclaim (%s) - there is no implicit prune-everything, and one call reclaims one kind", strings.Join(docker.PruneTargets, ", "))
	}
	if _, known := endpoints[target]; !known {
		return "", fmt.Errorf("unknown prune target %q: use one of %s", target, strings.Join(docker.PruneTargets, ", "))
	}
	if len(args.Prune) == 0 {
		return "", fmt.Errorf("not authorized: no prune: list in this tool's grant in mcp-sudo.yaml - list the targets it may reclaim (%s)", strings.Join(docker.PruneTargets, ", "))
	}
	if !docker.PruneAllowed(target, args.Prune) {
		return "", fmt.Errorf("not authorized to prune %s: it is not in this tool's prune: list in mcp-sudo.yaml (granted: %s)", target, strings.Join(args.Prune, ", "))
	}

	var res apiResult
	if err := args.Client(120 * time.Second).PostJSON(endpoints[target], nil, &res); err != nil {
		return "", err
	}
	deleted := res.ContainersDeleted
	deleted = append(deleted, res.VolumesDeleted...)
	deleted = append(deleted, res.NetworksDeleted...)
	deleted = append(deleted, res.CachesDeleted...)
	count := len(deleted)
	// Docker reports one entry per *event*, so a removed image usually
	// arrives twice - untagged, then deleted. Both are worth listing, but
	// only the deletions are counted, or one image reads as two.
	for _, img := range res.ImagesDeleted {
		if img.Deleted != "" {
			deleted = append(deleted, img.Deleted)
			count++
		} else if img.Untagged != "" {
			deleted = append(deleted, "untagged "+img.Untagged)
		}
	}
	result := targetResult{Target: target, Deleted: deleted, Count: count, Reclaimed: res.SpaceReclaimed}

	if docker.Structured(args.OutputFormat) {
		b, _ := json.Marshal(result)
		return string(b), nil
	}

	var text strings.Builder
	if target == "networks" {
		// /networks/prune is the only endpoint that reports no SpaceReclaimed
		// at all - "0.0 B" would read as a measurement.
		fmt.Fprintf(&text, "%s: %d removed\n", target, count)
	} else {
		fmt.Fprintf(&text, "%s: %d removed, %s reclaimed\n", target, count, humanSize(res.SpaceReclaimed))
	}
	for _, d := range deleted {
		fmt.Fprintf(&text, "  %s\n", short(d))
	}
	return text.String(), nil
}

// short trims the sha256: IDs Docker returns for deleted images and caches;
// names (containers, volumes, networks) are left alone.
func short(s string) string {
	// An untagged image is reported by name (nginx:old) or, once nothing tags
	// it, by digest - shorten the second kind too.
	if rest, ok := strings.CutPrefix(s, "untagged "); ok {
		return "untagged " + short(rest)
	}
	if strings.HasPrefix(s, "sha256:") && len(s) > 7+12 {
		return s[:7+12]
	}
	if len(s) == 64 {
		return s[:12]
	}
	return s
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
