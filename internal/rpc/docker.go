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

	var out []interface{}

	out = append(out, map[string]interface{}{
		"name":          "docker/containers",
		"tools_group":   "docker",
		"linuxctl_verb": "get",
		"description":   "Lists Docker containers through the Engine API on the local socket (no `docker` CLI needed): name, image, state, status, ports. Read-only. Only running containers unless `all: true`; `state` (created, running, paused, exited, ...) filters to one state and implies `all`; `pattern` is a glob on the container name (`web-*`); `limit` returns the newest N. Every container is listed: the grant's `containers:` list applies only to docker/manage, docker/logs and docker/exec. Text is a block per container plus a hint line, or `No containers found matching the criteria.`; `output_format: json` (also yaml/table/wide) returns an array of objects (name, names, id, full_id, image, image_id, command, state, status, created, ports, labels), `[]` when empty. For logs use `docker/logs`, to start or stop `docker/manage`, for one container's details the `docker-container://<name>/status` resource. Always runs as root: no `privileged` argument, refused unless the user's grant has `allowed: true`.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"all":           map[string]interface{}{"type": "boolean", "description": "Include stopped containers (docker ps -a)"},
				"pattern":       map[string]interface{}{"type": "string", "description": "Only containers whose name matches this glob (e.g. 'web-*')"},
				"state":         map[string]interface{}{"type": "string", "enum": []string{"created", "restarting", "running", "removing", "paused", "exited", "dead"}, "description": "Only containers in this state (implies all)"},
				"limit":         map[string]interface{}{"type": "integer", "description": "Return at most this many containers (newest first)"},
				"output_format": map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
			},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/manage",
		"tools_group":   "docker",
		"linuxctl_verb": "container",
		"description":   "Changes a container's lifecycle: start, stop, restart, kill, pause, unpause or remove; the container counterpart of `services/manage`. Mutating. Only containers in the grant's `containers:` list may be touched (name and ID are both checked; an empty list refuses everything). Always runs as root: no `privileged` argument, refused unless the grant has `allowed: true`. `stop` uses Docker's default 10 s grace. `remove` is irreversible for the container and deliberately fenced: it never sends `force` or volume removal, so a running container is refused (`stop or kill it first`) and anonymous volumes are kept. `start` on a running or `stop` on a stopped container reports `unchanged`, not an error. `container` is a name, full ID or ID prefix. Text: `Container NAME (ID12): stop succeeded.`; `output_format: json` returns container, id, action and result (`ok` or `unchanged`). For bulk cleanup use `docker/prune`, to run a command inside `docker/exec`, to check state first `docker/containers`.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"container":     map[string]interface{}{"type": "string", "description": "Container name, full ID or ID prefix"},
				"action":        map[string]interface{}{"type": "string", "enum": []string{"start", "stop", "restart", "kill", "pause", "unpause", "remove"}, "description": "Action to perform"},
				"output_format": map[string]interface{}{"type": "string", "description": "json returns container, id, action, result; default is text"},
			},
			"required": []string{"container", "action"},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/logs",
		"tools_group":   "docker",
		"linuxctl_verb": "logs",
		"description":   "Reads one container's recent logs, the container counterpart of `logs/journal-control`: stdout and stderr interleaved in Docker's order, the last `lines` lines (default 100), never a follow. Read-only. `since`/`until` take a unix timestamp (`1759005000`) or an RFC3339 time; `stdout` and `stderr` default to true (both false returns nothing); `timestamps` prefixes each line. The answer is capped at 1 MiB and an oversized tail is cut at the END without a marker, so the newest lines can be lost: ask for fewer `lines`. Only containers in the grant's `containers:` list; always runs as root, refused unless the grant has `allowed: true`. Plain log text by default (`Container X (ID) has no log output for this selection.` when empty); `output_format: json` returns container, id, lines, logs. To run a command use `docker/exec`; for host logs `logs/journal-control`.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"container":     map[string]interface{}{"type": "string", "description": "Container name, full ID or ID prefix"},
				"lines":         map[string]interface{}{"type": "integer", "description": "Tail this many lines (default 100)"},
				"since":         map[string]interface{}{"type": "string", "description": "Only entries after this time: unix timestamp (e.g. 1759005000) or RFC3339"},
				"until":         map[string]interface{}{"type": "string", "description": "Only entries before this time: unix timestamp or RFC3339"},
				"timestamps":    map[string]interface{}{"type": "boolean", "description": "Prefix every line with Docker's own timestamp"},
				"stdout":        map[string]interface{}{"type": "boolean", "description": "Include stdout (default true)"},
				"stderr":        map[string]interface{}{"type": "boolean", "description": "Include stderr (default true)"},
				"output_format": map[string]interface{}{"type": "string", "description": "json returns container, id, lines, logs; default is raw log text"},
			},
			"required": []string{"container"},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/exec",
		"tools_group":   "docker",
		"linuxctl_verb": "exec",
		"description":   "Runs ONE command inside a running container and returns its stdout, stderr and exit code: no TTY, no stdin, no follow. The highest-risk docker tool: arbitrary code, as the image's user (often root) unless `user` is set, with its own `containers:` grant list. Always runs as root at the Docker socket: no `privileged` argument, refused unless the grant has `allowed: true`. `command` is an argv array, not a shell line: `[\"sh\", \"-c\", \"ls /app\"]` for a shell. A non-zero exit code is not a tool error; read `exit_code`. `timeout` (seconds, default 30, capped at 300) is bounded by the worker limit, which is 30 s unless the operator sets `timeout_seconds` for docker/exec in daemon.yaml: then the answer is `timed out after N seconds` and the command keeps running in the container. Output is capped at 1 MiB. Text shows `Exit code: N` and the output; `output_format: json` returns container, id, command, exit_code, running, stdout, stderr. To read logs use `docker/logs`, to manage the container `docker/manage`.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"container":     map[string]interface{}{"type": "string", "description": "Container name, full ID or ID prefix"},
				"command":       map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "argv array, e.g. [\"sh\", \"-c\", \"ls /app\"]"},
				"user":          map[string]interface{}{"type": "string", "description": "Run as this user inside the container (name or uid[:gid]); default is the image's user"},
				"working_dir":   map[string]interface{}{"type": "string", "description": "Working directory inside the container"},
				"timeout":       map[string]interface{}{"type": "integer", "description": "Seconds to wait (default 30, values above 300 are capped at 300); the worker limit (30 s unless raised in daemon.yaml) ends the call first"},
				"output_format": map[string]interface{}{"type": "string", "description": "json returns container, id, command, exit_code, running, stdout, stderr; default is text"},
			},
			"required": []string{"container", "command"},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/images",
		"tools_group":   "docker",
		"linuxctl_verb": "images",
		"description":   "Lists Docker images through the Engine API on the local socket: tags, ID, size, creation time. Read-only (no pull, build or remove). Untagged intermediate layers are hidden unless `all: true`; untagged images that are shown appear as `<none>:<none>`. `pattern` is a glob on any tag or repository (`nginx*`, `*/api:*`). Text is tag lines with `ID | Size | Created`; `output_format: json` (also yaml/table/wide) returns an array of objects (id, full_id, repo_tags, repo_digests, size_bytes, created, containers = how many containers use it, labels), `[]` when empty. To delete unused images use `docker/prune` (target `images`, dangling only); for containers `docker/containers`; for one image's config the `docker-image://<ref>/inspect` resource. Always runs as root: no `privileged` argument, refused unless the grant has `allowed: true`.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"all":           map[string]interface{}{"type": "boolean", "description": "Include intermediate and untagged images (docker images -a)"},
				"pattern":       map[string]interface{}{"type": "string", "description": "Only images whose tag or repository matches this glob (e.g. 'nginx*', '*/api:*')"},
				"output_format": map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
			},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/volumes",
		"tools_group":   "docker",
		"linuxctl_verb": "volumes",
		"description":   "Lists Docker volumes with driver, mountpoint (a host path) and the names of the containers, running or stopped, that mount each. Read-only: volumes are never created or removed here. `pattern` is a glob on the volume name. Text is a block per volume; `output_format: json` (also yaml/table/wide) returns an array of objects (name, driver, mountpoint, created, scope, labels, options, in_use_by), `[]` when empty. `docker/prune` with target `volumes` removes anonymous unused volumes only. For one volume's details use the `docker-volume://<name>/inspect` resource, for containers `docker/containers`. Always runs as root: no `privileged` argument, refused unless the grant has `allowed: true`.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern":       map[string]interface{}{"type": "string", "description": "Only volumes whose name matches this glob"},
				"output_format": map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
			},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/networks",
		"tools_group":   "docker",
		"linuxctl_verb": "networks",
		"description":   "Lists Docker networks (not the host's: for host interfaces and routes use the `network://` resources and `network/*` tools) with driver, scope, subnet, gateway and the containers attached with their addresses. Read-only. `pattern` is a glob on the name; `driver` an exact driver (bridge, host, none, overlay, macvlan). Text shows `Attached: (nothing)` for an empty network; `output_format: json` (also yaml/table/wide) returns an array of objects (name, id, full_id, driver, scope, created, internal, attachable, ingress, ipam_driver, subnets, gateways, containers, options, labels). Removing unused networks is `docker/prune` (target `networks`); one network's full detail is the `docker-network://<name>/inspect` resource. Always runs as root: no `privileged` argument, refused unless the grant has `allowed: true`.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern":       map[string]interface{}{"type": "string", "description": "Only networks whose name matches this glob (e.g. 'app-*')"},
				"driver":        map[string]interface{}{"type": "string", "description": "Only networks using this driver (bridge, host, none, overlay, macvlan, ...)"},
				"output_format": map[string]interface{}{"type": "string", "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text"},
			},
		},
	})

	out = append(out, map[string]interface{}{
		"name":          "docker/prune",
		"tools_group":   "docker",
		"linuxctl_verb": "prune",
		"description":   "Deletes ONE kind of unused Docker object per call: `target` is containers (stopped), images (dangling only), volumes (anonymous, unused only; named volumes are never touched), networks (unused) or build-cache. Mutating and irreversible, and it deletes objects you did not name; call again for the next kind (pruning containers is what makes images dangling). The user's grant needs a `prune:` list containing the target, otherwise the call is refused before anything is deleted. Always runs as root: no `privileged` argument, refused unless the grant has `allowed: true`. The 30 s worker limit can end the call while the Engine keeps pruning. Text is `TARGET: N removed, X MiB reclaimed` plus the removed IDs (networks report no size); `output_format: json` returns target, deleted, count, space_reclaimed_bytes. To remove one named container use `docker/manage` (`remove`); to see what exists first `docker/containers`, `docker/images`, `docker/volumes`, `docker/networks`.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target": map[string]interface{}{
					"type":        "string",
					"enum":        docker.PruneTargets,
					"description": "The one kind to reclaim: containers (stopped), images (dangling), volumes (anonymous, unused), networks (unused) or build-cache",
				},
				"output_format": map[string]interface{}{"type": "string", "description": "json returns target, deleted, count, space_reclaimed_bytes; default is text"},
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
