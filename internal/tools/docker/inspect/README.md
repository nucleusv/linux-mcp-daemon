# docker/inspect (internal worker)

**Not a callable tool.** It is not in `tools/list` and `tools/call` will not route to it. It is the single internal worker behind every docker resource template, spawned by `internal/resources/templates/docker` — the same way `services/status` sits behind `service://{name}/status`.

One worker for all the schemes, selected by `kind`:

| `kind` | `view` | Engine API |
|---|---|---|
| `container` | `inspect` (default) | `GET /containers/<id>/json`, raw |
| `container` | `status` | same read, reduced to a computed summary |
| `container` | `stats` | `GET /containers/<id>/stats?stream=false` — one snapshot, never a stream |
| `container` | `top` | `GET /containers/<id>/top`, rendered as titles + rows |
| `image` | — | `GET /images/<ref>/json`, raw |
| `volume` | — | `GET /volumes/<name>`, raw |
| `network` | — | `GET /networks/<name>`, raw |

The `status` view exists because a raw inspect is several hundred lines of config; it computes what a person actually asks for — state, health, exit code, restart count, **uptime** (Docker reports `StartedAt`, not a duration), image, ports, networks and only those resource limits that are actually set.

## Usage & Permissions
A privileged resource read needs **two** grants (see `ARCHITECTURE.md`), and there is no unprivileged fallback here because the socket is root-owned:

1. the scheme in `resources:` — `docker-container://`, `docker-image://` or `docker-volume://`;
2. `docker/inspect: {allowed: true, containers: [...]}` in `tools:`.

For `kind: container` the grant's `containers:` list is passed in as `_containers` and enforced by the same `docker.Authorize` path `docker/manage` uses: resolve first, then match canonical name *and* ID, act on the ID.
