# FR-013 Docker networks: read-only listing and inspect

- **Created:** 2026-09-27, by the owner ("what about docker networks lets at least only readview")
- **Related:** FR-011 (Docker container management - same socket, same grant machinery), FR-012 (prune, which *can* remove networks), `internal/rpc/resources.go` (the `network://` scheme this must not collide with)

## Description

Read-only visibility into Docker networks: list them, and inspect one. No create, no remove, no connect/disconnect - a container's network membership is already visible from `container://{name}/inspect` in FR-011, and removing a network is prune's job (FR-012).

Read-only is the whole point of this ticket. Everything destructive about networks is deliberately elsewhere, so this one can be granted to an agent that should see the topology without being able to change it.

## The scheme collision - decide this before any code

`network://` is **already taken by host networking** and cannot be reused for Docker networks. Today `internal/rpc/resources.go` registers:

- `network://interfaces` (resource)
- `network://routes` (resource)
- `network://interfaces/{name}` (template)

and dispatches with `strings.HasPrefix(params.URI, "network://interfaces")` at `resources.go:240`. A `network://{name}/inspect` template for Docker would sit in the same namespace as `network://interfaces/{name}`, be ambiguous to read (`network://interfaces` = a host interface or a Docker network *named* "interfaces"?), and the existing prefix dispatch would shadow it.

**Decision: `docker-network://{name}/inspect`.** The one docker scheme that needs a prefix, because its bare noun is spoken for. `container://`, `image://` and `volume://` in FR-011 stay unprefixed - they collide with nothing, and renaming them for symmetry would be churn for its own sake.

Record in the docs that the asymmetry is deliberate, or someone will "fix" it later.

## Security model

Same as FR-011, minus the write surface:

- **Requires `privileged: true`** like every docker/* tool - the `docker`-group-is-root-equivalent argument in FR-011 applies unchanged, and it applies to reads too: network inspect exposes subnets, gateways, and every attached container's IP and alias.
- Gated by a `resources:` grant + `CanReadResourceAsRoot`, served by the same internal `docker/inspect` worker FR-011 introduces (no second worker).
- No `containers:`-style allowlist. There is nothing to scope: the tool cannot change anything, and a per-network allowlist would be granting "you may see this subnet but not that one", which is not a boundary worth the config surface. If that turns out to be wanted, it is a follow-up, not this ticket.

```yaml
resources:
  docker-network://: ""     # any docker network
```

## Proposed shape

- `docker/networks` - list networks: name, ID, driver, scope, subnet(s), gateway, internal/attachable flags, attached container count. `GET /networks`.
- `docker-network://{name}/inspect` - `GET /networks/{name}`: full config, IPAM, options, labels, and the `Containers` map (each attached container's name, IPv4/IPv6 address, MAC, endpoint ID).

No `/status` - like a volume, a network has no runtime state worth computing separately from its config.

## Depends on FR-011

This reuses FR-011's socket client, its `privileged: true` gate and its `docker/inspect` internal worker. Implement it after FR-011, or fold it into that branch - it is two endpoints, and shipping it separately means touching the same registry files twice.

## Acceptance criteria

- [ ] `docker/networks` implemented against `GET /networks` over the unix socket; no `docker` CLI binary invoked.
- [ ] `docker-network://{name}/inspect` template implemented against `GET /networks/{name}`, served by FR-011's internal `docker/inspect` worker (no new worker).
- [ ] Scheme is `docker-network://`, **not** `network://`; a `resources/list` and `resources/templates/list` read shows no ambiguity with `network://interfaces`, `network://routes` or `network://interfaces/{name}`.
- [ ] Both refuse outright without `privileged: true` granted for that specific tool/resource - never falls back to the caller's plain uid.
- [ ] Nothing in this ticket can create, remove, connect or disconnect a network; the tool set is read-only by construction, not by a flag.
- [ ] `configs/mcp-sudo.yaml` `privileged` reference block lists `docker/networks` and `docker-network://` (plus the `docker/inspect` worker grant, if FR-011 has not already added it).
- [ ] Socket missing/not responding → the same clear named error FR-011 defines.
- [ ] Docs: tool page, resource page, overview list, `command-reference.md`, man page; the `docker-network://` prefix asymmetry explained where the schemes are documented.
- [ ] `bash scripts/check_docs.sh` and `bash scripts/check_readmes.sh` pass.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | List | Go unit test against a fake Docker API (httptest unix-socket server) | networks with name/driver/scope/subnet/gateway | | |
| T2 | Inspect | Go unit test, canned `GET /networks/{name}` response | IPAM + `Containers` map with per-container IP/MAC | | |
| T3 | No scheme collision | `ParseSudoConfig` + resource dispatch unit test: read `network://interfaces`, `network://interfaces/eth0`, `docker-network://bridge` | each routes to its own handler; a docker network named `interfaces` or `routes` is still reachable only via `docker-network://` | | |
| T4 | No unprivileged fallback | Go unit test: no `privileged: true` grant → refused before any socket dial | "not authorized", no request sent | | |
| T5 | Read-only | Review + test: no code path issues POST/DELETE to `/networks*` | only GET requests reach the socket | | |
| T6 | Socket missing | Point `daemon.yaml` at a nonexistent socket | clear named error | | |
| T7 | Live: list + inspect | VPS 9091 with Docker, real networks | matches `docker network ls` / `docker network inspect` | | |
| T8 | Live: boundary | read `docker-network://` without the grant | refused, logged as denied | | |
| T9 | All deployments | local k8s, VPS systemd 9091, VPS Docker 9092 | consistent; AmneziaVPN's networks visible but untouched | | |
| T10 | Docs | `check_docs.sh`, `check_readmes.sh` | pass | | |

## Comments

- 2026-09-27 - created at the owner's request, read-only on purpose ("lets at least only readview"). Found and resolved before any code: `network://` is already the host-networking scheme (`network://interfaces`, `network://routes`, `network://interfaces/{name}`, with prefix dispatch at `internal/rpc/resources.go:240`), so Docker networks get `docker-network://` - the one prefixed docker scheme, deliberately asymmetric with FR-011's `container://`/`image://`/`volume://`. Depends on FR-011's socket client and `docker/inspect` worker; cheapest to land in the same branch.
