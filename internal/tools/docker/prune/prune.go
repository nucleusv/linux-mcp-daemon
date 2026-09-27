// Package prune reclaims unused Docker objects - the one docker/* tool that
// deletes things it was never told the names of.
//
// Every other destructive docker tool is scoped by container name
// (docker/manage's containers: list). Prune cannot be: it is scoped by what
// kind of garbage to collect, so it carries its own allowlist (`prune:` in
// mcp-sudo.yaml) and each requested target is checked against it before the
// socket is dialled at all.
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
	Targets      []string `json:"targets,omitempty"`
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
	Error     string   `json:"error,omitempty"`
}

func Prune(argsJSON []byte) (string, error) {
	var args Args
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}
	if len(args.Targets) == 0 {
		return "", fmt.Errorf("targets is required: name what to reclaim (%s) - there is no implicit prune-everything", strings.Join(docker.PruneTargets, ", "))
	}
	if len(args.Prune) == 0 {
		return "", fmt.Errorf("not authorized: no prune: list in this tool's grant in mcp-sudo.yaml - list the targets it may reclaim (%s)", strings.Join(docker.PruneTargets, ", "))
	}

	// Check every requested target before touching the socket, so a request
	// that is partly unauthorized deletes nothing at all.
	wanted := map[string]bool{}
	for _, t := range args.Targets {
		t = strings.ToLower(strings.TrimSpace(t))
		if _, known := endpoints[t]; !known {
			return "", fmt.Errorf("unknown prune target %q: use one or more of %s", t, strings.Join(docker.PruneTargets, ", "))
		}
		if !docker.PruneAllowed(t, args.Prune) {
			return "", fmt.Errorf("not authorized to prune %s: it is not in this tool's prune: list in mcp-sudo.yaml (granted: %s)", t, strings.Join(args.Prune, ", "))
		}
		wanted[t] = true
	}

	c := args.Client(120 * time.Second)
	var results []targetResult
	var firstErr error
	// docker.PruneTargets order, not the caller's: pruning containers first
	// is what makes their images dangling and their volumes unused, so one
	// call reclaims what two calls in the wrong order would miss.
	for _, target := range docker.PruneTargets {
		if !wanted[target] {
			continue
		}
		var res apiResult
		if err := c.PostJSON(endpoints[target], nil, &res); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			results = append(results, targetResult{Target: target, Error: err.Error()})
			continue
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
		results = append(results, targetResult{
			Target:    target,
			Deleted:   deleted,
			Count:     count,
			Reclaimed: res.SpaceReclaimed,
		})
	}

	// Every target failed: nothing was reclaimed, so this is a failed call,
	// not a report. A partial failure still reports what was deleted.
	if firstErr != nil && len(results) == countFailed(results) {
		return "", firstErr
	}

	var total int64
	for _, r := range results {
		total += r.Reclaimed
	}

	if docker.Structured(args.OutputFormat) {
		b, _ := json.Marshal(map[string]interface{}{
			"results":                     results,
			"total_space_reclaimed_bytes": total,
		})
		return string(b), nil
	}

	var text strings.Builder
	for _, r := range results {
		if r.Error != "" {
			fmt.Fprintf(&text, "%s: failed - %s\n", r.Target, r.Error)
			continue
		}
		if r.Target == "networks" {
			// /networks/prune is the only endpoint that reports no
			// SpaceReclaimed at all - "0.0 B" would read as a measurement.
			fmt.Fprintf(&text, "%s: %d removed\n", r.Target, r.Count)
		} else {
			fmt.Fprintf(&text, "%s: %d removed, %s reclaimed\n", r.Target, r.Count, humanSize(r.Reclaimed))
		}
		for _, d := range r.Deleted {
			fmt.Fprintf(&text, "  %s\n", short(d))
		}
	}
	fmt.Fprintf(&text, "\nTotal reclaimed: %s\n", humanSize(total))
	return text.String(), nil
}

func countFailed(results []targetResult) int {
	n := 0
	for _, r := range results {
		if r.Error != "" {
			n++
		}
	}
	return n
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
