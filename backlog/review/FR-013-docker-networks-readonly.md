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

- [x] `docker/networks` implemented against `GET /networks` over the unix socket; no `docker` CLI binary invoked.
- [x] `docker-network://{name}/inspect` template implemented against `GET /networks/{name}`, served by FR-011's internal `docker/inspect` worker (no new worker).
- [x] Scheme is `docker-network://`, **not** `network://`; a `resources/list` and `resources/templates/list` read shows no ambiguity with `network://interfaces`, `network://routes` or `network://interfaces/{name}`.
- [x] Both refuse outright without `privileged: true` granted for that specific tool/resource - never falls back to the caller's plain uid.
- [x] Nothing in this ticket can create, remove, connect or disconnect a network; the tool set is read-only by construction, not by a flag.
- [x] `configs/mcp-sudo.yaml` `privileged` reference block lists `docker/networks` and `docker-network://` (plus the `docker/inspect` worker grant, if FR-011 has not already added it).
- [x] Socket missing/not responding → the same clear named error FR-011 defines.
- [x] Docs: tool page, resource page, overview list, `command-reference.md`, man page; the `docker-network://` prefix asymmetry explained where the schemes are documented.
- [x] `bash scripts/check_docs.sh` and `bash scripts/check_readmes.sh` pass.
- [x] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | List | Go unit test against a fake Docker API (httptest unix-socket server) | networks with name/driver/scope/subnet/gateway | 2026-09-27, `golang:1.26` container, repo tree | ✅ PASS - `TestList`, `TestFilters` |
| T2 | Inspect | Go unit test, canned `GET /networks/{name}` response | IPAM + `Containers` map with per-container IP/MAC | 2026-09-27, `golang:1.26` | ✅ PASS - `TestNetworkInspect` (`internal/tools/docker/inspect`) |
| T3 | No scheme collision | `ParseSudoConfig` + resource dispatch unit test: read `network://interfaces`, `network://interfaces/eth0`, `docker-network://bridge` | each routes to its own handler; a docker network named `interfaces` or `routes` is still reachable only via `docker-network://` | 2026-09-27, `golang:1.26` + live registry on VPS 9092 | ✅ PASS - `TestNoSchemeCollisionWithHostNetworking`, `TestURIParsing`; live listing shows the four schemes distinctly |
| T4 | No unprivileged fallback | Go unit test: no `privileged: true` grant → refused before any socket dial | "not authorized", no request sent | 2026-09-27, `golang:1.26` | ✅ PASS - `TestPrepareDockerCallForcesRootOrRefuses`, `TestBothGrantsRequired`, `TestSchemeGrantIsScoped` |
| T5 | Read-only | Review + test: no code path issues POST/DELETE to `/networks*` | only GET requests reach the socket | 2026-09-27, `golang:1.26` | ✅ PASS - `TestOnlyReads` (fake API fails the test on any non-GET) |
| T6 | Socket missing | Point `daemon.yaml` at a nonexistent socket | clear named error | 2026-09-27, local mcpd, `docker_socket: /dind/nope.sock` | ✅ PASS - tool and resource both name the path (output below) |
| T7 | Live: list + inspect | VPS 9091 with Docker, real networks | matches `docker network ls` / `docker network inspect` | 2026-09-27, VPS 9092 (containerized), user `privileged` | ✅ PASS - same 4 networks/IDs/drivers/scopes as `docker network ls`, same 3 containers with identical IPv4 + MAC as `docker network inspect bridge` |
| T8 | Live: boundary | read `docker-network://` without the grant | refused, logged as denied | 2026-09-27, local mcpd, unprivileged user | ✅ PASS - refused by name, no socket dial (output below) |
| T9 | All deployments | local k8s, VPS systemd 9091, VPS Docker 9092 | consistent; AmneziaVPN's networks visible but untouched | 2026-09-27 | ✅ PASS with one caveat - see below |
| T10 | Docs | `check_docs.sh`, `check_readmes.sh` | pass | 2026-09-27, repo tree | ✅ PASS - `tools (45) / resources (11) / resource templates (10)`, all READMEs present |

### T6 - socket missing (verbatim)

Tool:

```text
docker socket /dind/nope.sock not found - is Docker installed and running on this host?
(mcpd never installs it; set worker.docker_socket in daemon.yaml for a non-standard path)
```

Resource (`docker-network://appnet/inspect`): the same text, wrapped as `Error: … (Code: -32603)`.

### T7 - live list + inspect vs the docker CLI (VPS 9092, verbatim)

```text
$ linuxctl get docker networks
host (f4d6a1a927a6)
  Driver: host | Scope: local
  Attached: mcpd-docker

bridge (af463e539438)
  Driver: bridge | Scope: local
  Subnet: 172.17.0.0/16 | Gateway: 172.17.0.1
  Attached: amnezia-awg2 (172.17.0.2), fr011-logger (172.17.0.3), fr011-oneshot, fr011-probe (172.17.0.4)

amnezia-dns-net (8eb660fd7d5f)
  Driver: bridge | Scope: local
  Subnet: 172.29.172.0/24 | Gateway: 172.29.172.1
  Attached: amnezia-awg2 (172.29.172.2)

none (c7b63c38a6e3)
  Driver: null | Scope: local
  Attached: (nothing)

Hint: For one network's IPAM, options and per-container addresses, read docker-network://<name>/inspect
```

Cross-check on the same host, same minute:

```text
$ docker network ls
NETWORK ID     NAME              DRIVER    SCOPE
8eb660fd7d5f   amnezia-dns-net   bridge    local
af463e539438   bridge            bridge    local
f4d6a1a927a6   host              host      local
c7b63c38a6e3   none              null      local

$ docker network inspect bridge --format '{{range .Containers}}{{.Name}} {{.IPv4Address}} {{.MacAddress}}{{println}}{{end}}'
amnezia-awg2 172.17.0.2/16 4e:6a:b8:fc:89:83
fr011-logger 172.17.0.3/16 a2:b5:bb:cc:f9:f9
fr011-probe 172.17.0.4/16 ce:58:89:38:c4:3e
```

Identical on every field. `fr011-oneshot` appears in the tool's `Attached:` list without an address because it is stopped - the container list reports the membership, the network's own endpoint is gone. The template read (`docker-network://bridge/inspect`) returned the full IPAM/options/labels/`Containers` document with the same three MACs, plus `Status.IPAM.Subnets["172.17.0.0/16"] = {IPsInUse: 6, DynamicIPsAvailable: 65530}`.

### T3 - live registry, the four network schemes side by side (VPS 9092, verbatim)

```text
  network://interfaces - Network interfaces, assigned IP addresses, and detailed RX/TX traffic statistics for all interfaces.
  network://routes     - IPv4 Routing Table (/proc/net/route). Hint: Use network/ping to test reachability.
  network://interfaces/{name} - Detailed properties and RX/TX traffic statistics of a specific network interface.
  docker-network://{name}/inspect - Full configuration of one Docker network: driver, scope, IPAM (subnets, gateways, IP ranges), options, labels and every attached container's name, IPv4/IPv6 address and MAC. …
```

### T8 - boundary refusal (verbatim)

```text
Error: reading docker-network://appnet/inspect needs a "docker-network://" grant in this user's
resources: in mcp-sudo.yaml (the Docker socket is root-owned, so these reads have no
unprivileged mode) (Code: -32603)
```

### T9 - the three deployments

| Target | Version | docker/networks | Note |
|---|---|---|---|
| local k8s (`linux-mcp-daemon` ns, 9091) | `0.3.5-dev-fr013` | socket-absent message | correct: the node runs containerd, there is no `/var/run/docker.sock`. The *consistent* answer here is the named error, not a listing. |
| VPS systemd 9091 | `0.3.5-dev-fr013` | code + grants deployed, not swept | its three users authenticate by salted hash and the owner holds the tokens; the `fr011` user now carries `docker/networks` and `docker-network://` (restarted 13:16Z). Live sweep needs the owner's token. |
| VPS Docker 9092 (`mcpd-docker`, containerized) | `0.3.5-dev-fr013` | ✅ full sweep, T7 above | `worker.docker_socket` moved off the `/proc/1/root/run/docker.sock` workaround to plain `/var/run/docker.sock`, which is what proves the containerized `dialPath` resolution live. |

AmneziaVPN's two networks (`bridge` membership and `amnezia-dns-net`) are visible in the listing and were only ever read - no tool in this ticket can connect, disconnect or remove anything.

## Comments

- 2026-09-27 - created at the owner's request, read-only on purpose ("lets at least only readview"). Found and resolved before any code: `network://` is already the host-networking scheme (`network://interfaces`, `network://routes`, `network://interfaces/{name}`, with prefix dispatch at `internal/rpc/resources.go:240`), so Docker networks get `docker-network://` - the one prefixed docker scheme, deliberately asymmetric with FR-011's `container://`/`image://`/`volume://`. Depends on FR-011's socket client and `docker/inspect` worker; cheapest to land in the same branch.
- 2026-09-27 - `git mv new/ → in-progress/`; implemented on FR-011's branch as planned (one new tool package `internal/tools/docker/networks`, one template kind in `internal/resources/templates/docker`, no new worker).
- 2026-09-27 - `GET /networks` returns an empty `Containers` map however many containers are attached, so membership is assembled from `GET /containers/json?all=true` - the same `volumeUsers` pattern `docker/volumes` already uses. Only an *inspect* fills `Containers`, which is why the listing and the template differ in what they can show (the listing has no MAC, no endpoint ID, and a stopped container appears with no address).
- 2026-09-27 - `worker.docker_socket` on VPS 9092 moved off the `/proc/1/root/run/docker.sock` workaround to plain `/var/run/docker.sock`; the containerized worker resolves it through the host mount namespace. Container recreated (not restarted - `docker_socket` is applied on start, not on reload).
- 2026-09-27 - out of scope, found while testing, needs its own ticket (FR-015): the linuxctl verb resolver silently runs a group's bare-reachable tool when the target keyword matches no granted tool verb, and its Case B never consults `reg.Templates`. So `get docker network appnet`, `get docker volume app-data` and `get docker container web-1 status` all run `docker/containers` with `Warning: N extra argument(s) ignored: …` instead of resolving to the matching resource template. Nothing in FR-013 works around it; `linuxctl resource docker-network://<name>/inspect` is the documented form and works.
- 2026-09-27 - all criteria met, all ten tests run with the output above; `git mv in-progress/ → review/`. Not closed - only the owner closes a ticket.
- 2026-09-29 - review pass: 10/10 criteria checked, 10/10 tests pass. The one gap in T9 - VPS systemd 9091 "code + grants deployed, not swept" - is closed: on the current build, `linuxctl get docker networks` (list, shows amnezia-awg2 attached to the bridge, read-only) and `get docker network bridge` (full inspect JSON) both work on 9091 as fr011 and on 9092 as privileged. Owner instruction: move to closed/ after the v0.4.0 release (FR-019).
