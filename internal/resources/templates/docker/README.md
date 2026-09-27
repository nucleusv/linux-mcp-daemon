# docker resource templates

Handler for the three docker templates:

- `container://{name}/{view}` — `status` (computed summary), `inspect`, `stats`, `top`
- `image://{name}/inspect`
- `volume://{name}/inspect`

All three spawn one privileged worker, `docker/inspect`, the way `service://{name}/status` spawns `services/status`.

## Parsing
A container name never contains `/`, so the view is split off the right and defaults to `inspect`. An image reference legitimately does contain `/` (`ghcr.io/org/api:v1`), so for `image://` and `volume://` only a trailing `/inspect` is trimmed and the rest is the name.

## Permissions
Both grants ARCHITECTURE.md describes, with no unprivileged fallback to drop to — the Docker socket is root-owned:

1. the scheme in the user's `resources:` (`container://`, `image://`, `volume://`);
2. `docker/inspect: {allowed: true, containers: [...]}` in the user's `tools:`.

The second grant's `containers:` list is what scopes `container://` reads, exactly as it scopes `docker/manage`. The master only injects `_containers` and `_docker_socket`; the worker resolves the name and enforces the list.
