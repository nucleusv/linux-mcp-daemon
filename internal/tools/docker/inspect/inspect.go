// Package inspect is the single internal worker behind every docker resource
// template: container://{name}/{status,inspect,stats,top}, image://{name}/inspect
// and volume://{name}/inspect. One worker rather than one per scheme, the same
// way services/status serves service://{name}/status.
//
// It is not registered as a callable tool; it is spawned by the template
// handlers in internal/resources/templates/docker.
package inspect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
)

type Args struct {
	docker.CommonArgs
	// Kind is container, image, volume or network.
	Kind string `json:"kind,omitempty"`
	// Name is the object to inspect.
	Name string `json:"name,omitempty"`
	// View selects what to return for a container: inspect (raw), status
	// (computed summary), stats (one snapshot) or top (processes).
	View string `json:"view,omitempty"`
}

// Inspect answers one template read, as indented JSON.
func Inspect(argsJSON []byte) (string, error) {
	out, err := read(argsJSON)
	if err != nil {
		return "", err
	}
	return indent(out), nil
}

// indent pretty-prints the answer. Every other resource template returns
// indented JSON (service://{name}/status, process://{pid}/{target}), while
// Docker's own inspect bodies arrive as a single line - several hundred fields
// on one line, which neither a reader nor an agent quoting one field back can
// follow. A body that is somehow not JSON is passed through untouched.
func indent(s string) string {
	var buf bytes.Buffer
	if json.Indent(&buf, []byte(s), "", "  ") != nil {
		return s
	}
	return buf.String()
}

func read(argsJSON []byte) (string, error) {
	var args Args
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	c := args.Client(30 * time.Second)

	switch args.Kind {
	case "container":
		// The containers: allowlist covers template reads too, so the same
		// Authorize path runs here: resolve first, then match name and ID.
		id, _, err := docker.Authorize(c, args.Name, args.Containers)
		if err != nil {
			return "", err
		}
		switch args.View {
		case "", "inspect":
			return raw(c, "/containers/"+id+"/json")
		case "status":
			return status(c, id)
		case "stats":
			// stream=false makes this one snapshot, not an open stream.
			return raw(c, "/containers/"+id+"/stats?stream=false")
		case "top":
			return top(c, id)
		default:
			return "", fmt.Errorf("unknown container view %q: use inspect, status, stats or top", args.View)
		}

	case "image":
		if err := docker.ValidImageRef(args.Name); err != nil {
			return "", err
		}
		return raw(c, "/images/"+args.Name+"/json")

	case "volume":
		if err := docker.ValidIdentifier(args.Name); err != nil {
			return "", err
		}
		return raw(c, "/volumes/"+args.Name)

	case "network":
		if err := docker.ValidIdentifier(args.Name); err != nil {
			return "", err
		}
		return raw(c, "/networks/"+args.Name)

	default:
		return "", fmt.Errorf("unknown kind %q: use container, image, volume or network", args.Kind)
	}
}

// raw passes the Engine API's own JSON through unchanged - for /inspect,
// /stats and the volume and image reads, where the full object is the point.
func raw(c *docker.Client, path string) (string, error) {
	b, err := c.GetRaw(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// top renders GET /containers/{id}/top - Docker's own Titles + Processes
// arrays - as a table, which is what `docker top` shows.
func top(c *docker.Client, id string) (string, error) {
	var answer struct {
		Titles    []string   `json:"Titles"`
		Processes [][]string `json:"Processes"`
	}
	if err := c.GetJSON("/containers/"+id+"/top", &answer); err != nil {
		return "", err
	}
	rows := make([]map[string]string, 0, len(answer.Processes))
	for _, p := range answer.Processes {
		row := map[string]string{}
		for i, title := range answer.Titles {
			if i < len(p) {
				row[strings.ToLower(title)] = p[i]
			}
		}
		rows = append(rows, row)
	}
	b, _ := json.Marshal(map[string]interface{}{
		"titles":    answer.Titles,
		"processes": rows,
	})
	return string(b), nil
}

// inspectAnswer is the subset of a container inspect the computed status view
// is built from.
type inspectAnswer struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	State struct {
		Status     string `json:"Status"`
		Running    bool   `json:"Running"`
		Paused     bool   `json:"Paused"`
		Restarting bool   `json:"Restarting"`
		OOMKilled  bool   `json:"OOMKilled"`
		Dead       bool   `json:"Dead"`
		Pid        int    `json:"Pid"`
		ExitCode   int    `json:"ExitCode"`
		Error      string `json:"Error"`
		StartedAt  string `json:"StartedAt"`
		FinishedAt string `json:"FinishedAt"`
		Health     *struct {
			Status        string `json:"Status"`
			FailingStreak int    `json:"FailingStreak"`
		} `json:"Health"`
	} `json:"State"`
	Created      string `json:"Created"`
	RestartCount int    `json:"RestartCount"`
	Config       struct {
		Image  string   `json:"Image"`
		Cmd    []string `json:"Cmd"`
		Labels map[string]string
	} `json:"Config"`
	HostConfig struct {
		Memory        int64  `json:"Memory"`
		MemorySwap    int64  `json:"MemorySwap"`
		NanoCpus      int64  `json:"NanoCpus"`
		CpuShares     int64  `json:"CpuShares"`
		PidsLimit     *int64 `json:"PidsLimit"`
		Privileged    bool   `json:"Privileged"`
		RestartPolicy struct {
			Name              string `json:"Name"`
			MaximumRetryCount int    `json:"MaximumRetryCount"`
		} `json:"RestartPolicy"`
	} `json:"HostConfig"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
			Gateway   string `json:"Gateway"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

// status is the computed summary that earns container://{name}/status a place
// next to /inspect: state, health, exit code, restart count, uptime, image,
// ports and resource limits, rather than several hundred lines of raw config.
func status(c *docker.Client, id string) (string, error) {
	var in inspectAnswer
	if err := c.GetJSON("/containers/"+id+"/json", &in); err != nil {
		return "", err
	}

	out := map[string]interface{}{
		"name":           strings.TrimPrefix(in.Name, "/"),
		"id":             in.ID,
		"state":          in.State.Status,
		"running":        in.State.Running,
		"paused":         in.State.Paused,
		"restarting":     in.State.Restarting,
		"oom_killed":     in.State.OOMKilled,
		"exit_code":      in.State.ExitCode,
		"pid":            in.State.Pid,
		"restart_count":  in.RestartCount,
		"image":          in.Config.Image,
		"created":        in.Created,
		"started_at":     in.State.StartedAt,
		"privileged":     in.HostConfig.Privileged,
		"restart_policy": policy(in),
		"limits":         limits(in),
		"ports":          ports(in),
		"networks":       networks(in),
	}
	if in.State.Error != "" {
		out["error"] = in.State.Error
	}
	if in.State.Health != nil {
		out["health"] = in.State.Health.Status
		out["health_failing_streak"] = in.State.Health.FailingStreak
	}
	if u := uptime(in); u != "" {
		out["uptime"] = u
	}
	if !in.State.Running && in.State.FinishedAt != "" {
		out["finished_at"] = in.State.FinishedAt
	}

	b, _ := json.Marshal(out)
	return string(b), nil
}

// uptime is how long the container has been up, computed from StartedAt -
// Docker reports the timestamp, not the duration.
func uptime(in inspectAnswer) string {
	if !in.State.Running || in.State.StartedAt == "" {
		return ""
	}
	started, err := time.Parse(time.RFC3339Nano, in.State.StartedAt)
	if err != nil {
		return ""
	}
	return time.Since(started).Round(time.Second).String()
}

func policy(in inspectAnswer) map[string]interface{} {
	p := in.HostConfig.RestartPolicy
	return map[string]interface{}{
		"name":                p.Name,
		"maximum_retry_count": p.MaximumRetryCount,
	}
}

// limits reports only the limits actually set; a map full of zeroes says
// "unlimited" far less clearly than the key being absent.
func limits(in inspectAnswer) map[string]interface{} {
	h := in.HostConfig
	out := map[string]interface{}{}
	if h.Memory > 0 {
		out["memory_bytes"] = h.Memory
	}
	if h.MemorySwap > 0 {
		out["memory_swap_bytes"] = h.MemorySwap
	}
	if h.NanoCpus > 0 {
		out["cpus"] = float64(h.NanoCpus) / 1e9
	}
	if h.CpuShares > 0 {
		out["cpu_shares"] = h.CpuShares
	}
	if h.PidsLimit != nil && *h.PidsLimit > 0 {
		out["pids_limit"] = *h.PidsLimit
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ports(in inspectAnswer) []string {
	var out []string
	for spec, bindings := range in.NetworkSettings.Ports {
		if len(bindings) == 0 {
			out = append(out, spec)
			continue
		}
		for _, b := range bindings {
			host := b.HostIP
			if host == "" {
				host = "0.0.0.0"
			}
			out = append(out, fmt.Sprintf("%s:%s->%s", host, b.HostPort, spec))
		}
	}
	return out
}

func networks(in inspectAnswer) map[string]string {
	if len(in.NetworkSettings.Networks) == 0 {
		return nil
	}
	out := map[string]string{}
	for name, n := range in.NetworkSettings.Networks {
		out[name] = n.IPAddress
	}
	return out
}
