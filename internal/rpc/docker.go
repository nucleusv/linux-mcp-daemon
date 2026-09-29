package rpc

// The docker/* tool schemas and the one place their calls are prepared.
//
// They live apart from tools.go for two reasons: the schema list there is
// already long, and every docker tool shares one authorization shape that is
// unlike the rest of the daemon's - root always, a containers: allowlist for
// the tools naming a container, and a socket path the worker cannot read for
// itself (worker mode never loads daemon.yaml).

import (
	"encoding/json"
	"fmt"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
)

// dockerTools returns every docker tool schema. Every user sees all of them:
// a tool the user's grant does not allow is refused when called (see
// prepareDockerCall), with an error naming the missing grant, rather than
// being left out of the listing where a wrong call would look like a typo.
func dockerTools() []interface{} {
	// Repeated in every docker schema: the tools are root-only by nature, so
	// there is no `privileged` parameter to pass - the daemon sets it.
	const rootNote = " This tool always runs as root (the Docker socket is root-owned), so it takes no `privileged` argument - a call is refused unless this user's grant in mcp-sudo.yaml allows it."
	const allowNote = " Which containers it may touch comes from the `containers:` list in this user's grant, not from the call."

	var out []interface{}

	out = append(out, map[string]interface{}{
		"name":          "docker/containers",
		"tools_group":   "docker",
		"linuxctl_verb": "get",
		"description":   "Lists Docker containers with image, state, status and ports (plus labels in `output_format: json`) - the Engine API's own view, read straight from /var/run/docker.sock (the `docker` CLI is never invoked and need not be installed). Running containers only by default; `all: true` includes exited and created ones, and `state` filters to one state. Hint: for one container's runtime summary read container://<name>/status, for its processes container://<name>/top." + rootNote,
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"all":           map[string]interface{}{"type": "boolean", "description": "Include stopped containers (docker ps -a)"},
				"pattern":       map[string]interface{}{"type": "string", "description": "Only containers whose name matches this glob (e.g. 'web-*')"},
				"state":         map[string]interface{}{"type": "string", "enum": []string{"created", "restarting", "running", "removing", "paused", "exited", "dead"}, "description": "Only containers in this state (implies all)"},
				"limit":         map[string]interface{}{"type": "integer", "description": "Return at most this many containers"},
				"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
			},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/manage",
		"tools_group":   "docker",
		"linuxctl_verb": "container",
		"description":   "Container lifecycle: start, stop, restart, kill, pause, unpause, remove - what services/manage is for systemd units." + allowNote + " `remove` is the one irreversible action and is deliberately fenced: it never sends Docker's `force` or `v`, so a running container is refused (stop or kill it first, two separate audited calls) and anonymous volumes are never deleted. Restarting a container that holds state, or removing one that is not reproducible from its image and volumes, loses work - check container://<name>/status first." + rootNote,
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"container":     map[string]interface{}{"type": "string", "description": "Container name, full ID or ID prefix"},
				"action":        map[string]interface{}{"type": "string", "enum": []string{"start", "stop", "restart", "kill", "pause", "unpause", "remove"}, "description": "Action to perform"},
				"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml). Defaults to text"},
			},
			"required": []string{"container", "action"},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/logs",
		"tools_group":   "docker",
		"linuxctl_verb": "logs",
		"description":   "Reads one container's logs - the shape logs/journal-control has for systemd units. Both streams interleaved in the order Docker recorded them (`stdout: false` or `stderr: false` drops one), last 100 lines unless `lines` says otherwise, and never a follow: this returns what is there and ends. Answers are capped at 1 MiB; ask for a smaller tail rather than a bigger answer." + allowNote + rootNote,
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"container":     map[string]interface{}{"type": "string", "description": "Container name, full ID or ID prefix"},
				"lines":         map[string]interface{}{"type": "integer", "description": "Tail this many lines (default 100)"},
				"since":         map[string]interface{}{"type": "string", "description": "Only entries after this time (unix timestamp, or RFC3339)"},
				"until":         map[string]interface{}{"type": "string", "description": "Only entries before this time (unix timestamp, or RFC3339)"},
				"timestamps":    map[string]interface{}{"type": "boolean", "description": "Prefix every line with Docker's own timestamp"},
				"stdout":        map[string]interface{}{"type": "boolean", "description": "Include stdout (default true)"},
				"stderr":        map[string]interface{}{"type": "boolean", "description": "Include stderr (default true)"},
				"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json). Defaults to raw log text"},
			},
			"required": []string{"container"},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/exec",
		"tools_group":   "docker",
		"linuxctl_verb": "exec",
		"description":   "Runs one command inside a running container and returns its stdout, stderr and exit code. No TTY, no interactive attach, no follow - one command, one answer, output capped at 1 MiB and a 30s default timeout (300s maximum; a timed-out exec keeps running inside the container, the Engine API has no way to cancel one). `command` is an argv array, not a shell line: [\"sh\", \"-c\", \"ls /app\"] to use a shell. This is the sharpest tool in the group - arbitrary code, as root by default, inside whatever it targets - so its `containers:` list is separate from every other docker tool's." + allowNote + rootNote,
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"container":     map[string]interface{}{"type": "string", "description": "Container name, full ID or ID prefix"},
				"command":       map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "argv array, e.g. [\"sh\", \"-c\", \"ls /app\"]"},
				"user":          map[string]interface{}{"type": "string", "description": "Run as this user inside the container (name or uid[:gid]); default is the image's user"},
				"working_dir":   map[string]interface{}{"type": "string", "description": "Working directory inside the container"},
				"timeout":       map[string]interface{}{"type": "integer", "description": "Seconds to wait for the command (default 30, maximum 300)"},
				"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json). Defaults to text"},
			},
			"required": []string{"container", "command"},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/images",
		"tools_group":   "docker",
		"linuxctl_verb": "images",
		"description":   "Lists Docker images with tags, size and creation time (plus digests, labels and how many containers use each in `output_format: json`). Read-only by construction: no pull, no build, no remove. Untagged intermediate layers are hidden unless `all: true`. Hint: for one image's full config read image://<ref>/inspect." + rootNote,
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"all":           map[string]interface{}{"type": "boolean", "description": "Include intermediate and untagged images (docker images -a)"},
				"pattern":       map[string]interface{}{"type": "string", "description": "Only images whose tag or repository matches this glob (e.g. 'nginx*', '*/api:*')"},
				"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
			},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/volumes",
		"tools_group":   "docker",
		"linuxctl_verb": "volumes",
		"description":   "Lists Docker volumes with driver, mountpoint and the containers currently mounting each (Docker reports only a refcount; the names are more useful, so this cross-references the container list). Read-only: no create, no remove - a volume is the one part of a Docker install that holds data nothing else can rebuild. Hint: for one volume's options and labels read volume://<name>/inspect." + rootNote,
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern":       map[string]interface{}{"type": "string", "description": "Only volumes whose name matches this glob"},
				"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
			},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/networks",
		"tools_group":   "docker",
		"linuxctl_verb": "networks",
		"description":   "Lists Docker networks with driver, scope, subnet, gateway and the containers attached to each with their addresses (plus IPAM options and labels in `output_format: json`). Read-only by construction: nothing here creates or removes a network, or connects a container to one - removing an unused network is docker/prune's job. Note the scheme for a single network is `docker-network://`, not `network://`: that one is already the host's own networking (network://interfaces, network://routes). Hint: for one network's full IPAM, options and per-container MAC addresses read docker-network://<name>/inspect." + rootNote,
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern":       map[string]interface{}{"type": "string", "description": "Only networks whose name matches this glob (e.g. 'app-*')"},
				"driver":        map[string]interface{}{"type": "string", "description": "Only networks using this driver (bridge, host, none, overlay, macvlan, ...)"},
				"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text"},
			},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/prune",
		"tools_group":   "docker",
		"linuxctl_verb": "prune",
		"description":   "Reclaims disk by deleting unused Docker objects: stopped containers, dangling images, anonymous volumes, unused networks, or the build cache. `target` names exactly one of those kinds and is required - there is no \"everything\", and one call reclaims one kind. Call it again for the next kind: the kinds are not independent (pruning containers is what makes their images dangling and their anonymous volumes unused), so reclaiming two in one call would delete more than either request described. This is the only docker tool that deletes objects it was never given the names of, so it is scoped by kind rather than by container: which kinds this user may reclaim comes from the `prune:` list in their grant, not from the call, and a target outside that list refuses before anything is deleted. Two Engine API defaults are never overridden: images prunes only *dangling* ones (an image a stopped container still references survives), and volumes only *anonymous* ones (a named volume, the one thing in a Docker install nothing can rebuild, is never touched)." + rootNote,
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target": map[string]interface{}{
					"type":        "string",
					"enum":        docker.PruneTargets,
					"description": "The one kind to reclaim: containers (stopped), images (dangling), volumes (anonymous, unused), networks (unused) or build-cache",
				},
				"output_format": map[string]interface{}{"type": "string", "description": "Desired output format (e.g. json, yaml). Defaults to text"},
			},
			"required": []string{"target"},
		},
	})

	return out
}

// prepareDockerCall authorizes a docker tool call and rewrites its arguments
// for the worker. It returns the arguments to spawn with.
//
// Three things happen here, all of them the daemon's business and none of them
// the caller's:
//
//   - the grant is checked. Without it the spawn would fail anyway, but with a
//     message about `privileged: true` that says nothing about Docker.
//   - `privileged: true` is forced, in the arguments as well as for the spawn,
//     so the audit line records what actually ran.
//   - `_docker_socket`, `_containers` and `_prune` are delete-then-set, so a
//     caller can never supply any of them. The socket comes from daemon.yaml,
//     which worker mode never reads; the two allowlists are the whole
//     authorization boundary for the tools that name a container and for
//     docker/prune, which names none.
func prepareDockerCall(sudoCfg *config.SudoConfig, user, tool string, args json.RawMessage) (json.RawMessage, error) {
	if !sudoCfg.CanRunAsRoot(user, tool) {
		return args, fmt.Errorf("user %s is not authorized to run %s: the Docker socket is root-owned, so every docker/* tool needs `allowed: true` in its grant in mcp-sudo.yaml (and, for the tools naming one container, a `containers:` list)", user, tool)
	}

	var argMap map[string]interface{}
	if err := json.Unmarshal(args, &argMap); err != nil || argMap == nil {
		argMap = map[string]interface{}{}
	}
	delete(argMap, "_docker_socket")
	delete(argMap, "_containers")
	delete(argMap, "_prune")
	argMap["privileged"] = true
	if docker.SocketPath != "" {
		argMap["_docker_socket"] = docker.SocketPath
	}
	if config.ContainerTools[tool] {
		// Empty (a grant without containers:) refuses everything in the worker,
		// before it dials the socket - see docker.Authorize.
		argMap["_containers"] = sudoCfg.GetAllowedContainers(user, tool)
	}
	if tool == config.PruneTool {
		argMap["_prune"] = sudoCfg.GetAllowedPruneTargets(user, tool)
	}
	b, err := json.Marshal(argMap)
	if err != nil {
		return args, err
	}
	return b, nil
}
