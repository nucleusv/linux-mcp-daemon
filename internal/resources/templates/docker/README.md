# docker resource templates

Handler for the four docker templates:

| Template | What it returns |
|---|---|
| `docker-container://{name}/{view}` | one container's own view of itself; `{view}` is `status` (computed summary: state, health, exit code, restart count, uptime, image, ports, limits), `inspect` (the full Engine API inspect), `stats` (one CPU/memory/network/block-IO snapshot, returned once, never streamed) or `top` (the container's processes); with no view it defaults to `inspect` |
| `docker-image://{name}/inspect` | full configuration of one image (layers, env, entrypoint, labels, digests); `{name}` may be a tag (`nginx:alpine`), a repository path (`ghcr.io/org/api:v1`) or an image ID |
| `docker-volume://{name}/inspect` | driver, mountpoint, options, labels and scope of one volume |
| `docker-network://{name}/inspect` | full configuration of one network: driver, scope, IPAM (subnets, gateways, IP ranges), options, labels and the containers attached to it |

`docker-network://` is the one prefixed scheme, on purpose: `network://` is already the host's own networking (`network://interfaces`, `network://routes`), so a Docker network cannot have the bare noun. The other three are unprefixed because nothing collides with them.

All four spawn one privileged worker, `docker/inspect`, the way `service://{name}/status` spawns `services/status`.

## Parsing
A container name never contains `/`, so the view is split off the right and defaults to `inspect`. An image reference legitimately does contain `/` (`ghcr.io/org/api:v1`), so for `docker-image://`, `docker-volume://` and `docker-network://` only a trailing `/inspect` is trimmed and the rest is the name. An unknown container view is refused with the list of valid ones.

## Permissions
Both grants ARCHITECTURE.md describes, with no unprivileged fallback to drop to - the Docker socket is root-owned:

1. the scheme in the user's `resources:` (`docker-container://`, `docker-image://`, `docker-volume://`, `docker-network://`);
2. `docker/inspect: {allowed: true, containers: [...]}` in the user's `tools:`.

The second grant's `containers:` list is what scopes `docker-container://` reads, exactly as it scopes `docker/manage`. The master only injects `_containers` and `_docker_socket`; the worker resolves the name and enforces the list.

Without either grant the template is still listed, but the read is refused with an error naming the missing grant.

## From `linuxctl`
`linuxctl describe docker container web-1` (or `get docker container web-1 status`), `get docker image nginx:alpine`, `get docker volume app-data`, `get docker network bridge`.
