// Package docker serves the container://, image://, volume:// and
// docker-network:// resource templates. All four spawn one internal worker,
// docker/inspect, the way service://{name}/status spawns services/status.
//
// A docker read is privileged or nothing - the socket is root-owned - so it
// needs both grants ARCHITECTURE.md describes: the scheme in `resources:`, and
// `docker/inspect` in `tools:` (whose `containers:` list scopes container://
// reads, exactly as it scopes docker/manage).
package docker

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
	"github.com/nucleusv/linux-mcp-daemon/internal/worker"
)

// containerViews are the four things container://{name}/<view> can return.
var containerViews = map[string]bool{"inspect": true, "status": true, "stats": true, "top": true}

// schemes maps each kind this handler serves to the resources: key that grants
// it. scripts/check_docs.sh reads these literals to know which grants the
// reference config must list.
// docker-network:// is the one prefixed scheme here, deliberately asymmetric
// with the other three: network:// is already the host's own networking
// (network://interfaces, network://routes), so a Docker network cannot have the
// bare noun. Renaming the other three for symmetry would be churn.
var schemes = map[string]string{
	"container":      "container://",
	"image":          "image://",
	"volume":         "volume://",
	"docker-network": "docker-network://",
}

// Handle reads one of the docker templates. uri is
// container://{name}/{status|inspect|stats|top}, image://{ref}/inspect or
// volume://{name}/inspect.
func Handle(uri string, sessionUser string, sudoConfig *config.SudoConfig) (string, string, error) {
	kind, rest, ok := strings.Cut(uri, "://")
	if !ok {
		return "", "", fmt.Errorf("unknown resource: %s", uri)
	}
	scheme, known := schemes[kind]
	if !known {
		return "", "", fmt.Errorf("unknown resource: %s", uri)
	}

	var name, view string
	switch kind {
	case "container":
		// Split off the view from the right: a container name never contains
		// "/", so anything after the last one is the view.
		name, view = rest, "inspect"
		if i := strings.LastIndex(rest, "/"); i >= 0 {
			name, view = rest[:i], rest[i+1:]
		}
		if !containerViews[view] {
			return "", "", fmt.Errorf("unknown view %q for %s: use container://%s/status, /inspect, /stats or /top", view, uri, name)
		}
	default: // image, volume, docker-network
		// An image reference legitimately contains "/" (ghcr.io/org/img), so
		// only the /inspect suffix is trimmed.
		name = strings.TrimSuffix(rest, "/inspect")
		view = "inspect"
	}
	if name == "" {
		return "", "", fmt.Errorf("no name in %s", uri)
	}

	// Grant 1: the scheme. Unlike file:// or process://, there is no
	// unprivileged fallback to drop to - the socket is root-owned.
	if !sudoConfig.CanReadResourceAsRoot(sessionUser, scheme, name) {
		return "", "", fmt.Errorf("reading %s needs a %q grant in this user's resources: in mcp-sudo.yaml (the Docker socket is root-owned, so these reads have no unprivileged mode)", uri, scheme)
	}
	// Grant 2: the worker behind the template.
	if !sudoConfig.CanRunAsRoot(sessionUser, "docker/inspect") {
		return "", "", fmt.Errorf("reading %s also needs `docker/inspect: {allowed: true, containers: [...]}` in this user's tools: in mcp-sudo.yaml - that is the internal worker these templates spawn, and its containers: list is what scopes container:// reads", uri)
	}

	// The worker's kinds are the plain nouns; only the URI scheme is prefixed.
	workerKind := strings.TrimPrefix(kind, "docker-")
	args := map[string]interface{}{"kind": workerKind, "name": name, "view": view}
	if docker.SocketPath != "" {
		args["_docker_socket"] = docker.SocketPath
	}
	if kind == "container" {
		args["_containers"] = sudoConfig.GetAllowedContainers(sessionUser, "docker/inspect")
	}
	argsJSON, _ := json.Marshal(args)

	content, readErr := worker.SpawnWorker(sessionUser, "docker/inspect", argsJSON, true, sudoConfig, 30)
	return content, "application/json", readErr
}
