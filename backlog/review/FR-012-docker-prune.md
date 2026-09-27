# FR-012 Docker prune: remove unused containers, images, volumes, networks, build cache

- **Created:** 2026-09-27, by the owner
- **Related:** FR-011 (Docker container management), `internal/tools/docker/manage` lifecycle actions

## Description

`docker/manage remove <container>` removes one named container. `docker prune` removes many unused objects at once: stopped containers, dangling/untagged images, unused volumes, unused networks, and the build cache. It is intentionally a separate tool from `docker/manage` because it cannot be scoped by container name — it is scoped by *what kind of garbage* to collect — and has a much wider blast radius.

Useful for: cleaning up after CI builds, reclaiming disk on a node, pruning stale images after deployments. Dangerous because: a misconfigured or over-eager prune can delete volumes that still hold wanted data, or remove images that would otherwise be reused.

## Security model

- Like all docker/* tools, **requires `privileged: true`** (no unprivileged fallback).
- Uses a separate `prune:` allowlist (mirrors `containers:` and `paths:`): the grant lists which targets may be pruned.
- `volumes` and `images` are the riskiest targets and should require explicit listing; `containers` (stopped) and `networks` are lower risk but still destructive.
- A grant with `allowed: true` and no `prune:` list is rejected in strict mode, same pattern as `containers:` in FR-011.
- Never prune everything (`all`) implicitly; the caller must name each target.

## Proposed tool shape

- `docker/prune`
  - `targets`: array of strings, one or more of `containers`, `images`, `volumes`, `networks`, `build-cache`
  - `filters`: optional Docker API prune filters (label, until, dangling) — start without filters, add if needed

Example grant:

```yaml
users:
  ci-agent:
    privileged:
      tools:
        docker/prune:
          allowed: true
          prune:
            - images        # may prune dangling/untagged images only
            - build-cache
```

## Acceptance criteria

- [x] `docker/prune` tool implemented against the Docker Engine API (`/containers/prune`, `/images/prune`, `/volumes/prune`, `/networks/prune`, `/build/prune`).
- [x] Requires `privileged: true`; no unprivileged fallback.
- [x] `prune:` allowlist enforced; strict config load rejects `allowed: true` without a non-empty `prune:` list.
- [x] Each target returns the count and reclaimed space (where the API provides it) or a clear message. `networks` is the one endpoint that reports no `SpaceReclaimed`, so its line omits the clause rather than printing `0.0 B`.
- [x] Audit-logged with the full target list and reclaimed summary.
- [x] `configs/mcp-sudo.yaml` reference `privileged` block updated with `docker/prune`.
- [x] Docs page and overview list updated (`docs/website/docs/mcp-api/tools/docker/prune.md`, `mcp-api/overview.md`, `configuration/mcp-sudo.md`, `linuxctl/command-reference.md`, `docs/man/linuxctl.1`).
- [x] Definition of Done (backlog/README.md).

**Scope decision:** `filters` was dropped. Docker's `--all`, `--force` and `--filter until=/label=` are deliberately not exposed, because the two unfiltered Engine API defaults *are* the safety margin: `/images/prune` without filters takes only dangling images (`dangling=false` would take every image no running container uses), and `/volumes/prune` without filters only anonymous ones (`all=true` would take named volumes, the one thing in a Docker install nothing can rebuild). The scoping knob is the grant's `prune:` list, not an argument the caller picks. The tool therefore sends no filters at all.

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Prune images | Go unit test against fake Docker API | hits `/images/prune`, returns count/reclaimed | 2026-09-27, macOS + golang:1.26 container | ✅ `TestPruneImages` PASS |
| T2 | Multiple targets | Go unit test | hits each requested endpoint | 2026-09-27, macOS + golang:1.26 container | ✅ `TestMultipleTargetsInDependencyOrder`, `TestPartialFailure` PASS |
| T3 | Not authorized target | Grant `prune: ["containers"]`, call with `targets: ["volumes"]` | refused before any API call | 2026-09-27, unit + live (`fr011-mcpd`, user `unpriviliged`) | ✅ `TestUnauthorizedTargetDeletesNothing` PASS; live refusal below |
| T4 | Missing prune list | `ParseSudoConfig` strict | load error | 2026-09-27, macOS + golang:1.26 container | ✅ `TestPruneTargetsRequired` PASS (`internal/config`) |
| T5 | Live: prune stopped containers | Docker-in-Docker host `fr011-dind` via `fr011-mcpd`, throwaway fixtures | objects removed, counts reported, named volume + in-use networks survive | 2026-09-27, local dind | ✅ output below; `fr012-named`, `app-data`, `appnet`, `backend`, `bridge`, `host`, `none` and all three running containers survived |
| T6 | Audit | daemon log | `docker/prune` call logged with targets and reclaimed summary | 2026-09-27, `fr011-mcpd` stderr | ✅ logged at every level |
| T7 | Docs | `check_docs.sh` | passes | 2026-09-27, macOS | ✅ tools 46, resources 11, templates 10; `check_readmes.sh` ✅ |

## Evidence

### T1, T2, T4 - unit tests (2026-09-27)

```
$ go test -v ./internal/tools/docker/prune/
=== RUN   TestPruneImages
--- PASS: TestPruneImages (0.00s)
=== RUN   TestMultipleTargetsInDependencyOrder
--- PASS: TestMultipleTargetsInDependencyOrder (0.00s)
=== RUN   TestUnauthorizedTargetDeletesNothing
--- PASS: TestUnauthorizedTargetDeletesNothing (0.00s)
=== RUN   TestPartialFailure
--- PASS: TestPartialFailure (0.00s)
PASS
ok  	github.com/nucleusv/linux-mcp-daemon/internal/tools/docker/prune	0.582s

$ go test -v -run Prune ./internal/config/
=== RUN   TestPruneTargetsRequired
--- PASS: TestPruneTargetsRequired (0.00s)
PASS
ok  	github.com/nucleusv/linux-mcp-daemon/internal/config	0.395s
```

`internal/rpc` can't build on macOS (`internal/worker/hostns.go` needs `syscall.Unshare`/`unix.Setns`), so the whole suite was run in a `golang:1.26` container against the same working tree - all green, including `internal/rpc`:

```
$ docker run --rm -v "$PWD":/src -w /src golang:1.26 sh -c 'go test ./... | grep -v "no test files"'
ok  	github.com/nucleusv/linux-mcp-daemon/internal/rpc	0.004s
...
ok  	github.com/nucleusv/linux-mcp-daemon/internal/tools/docker/prune	0.003s
```

(An alpine image fails `internal/fsafe/TestNoFollowNormalUse` - busybox `find` has no `-printf`. Environment artifact, not a regression; Debian-based image passes.)

### T3 - refusals, live against `fr011-mcpd` (2026-09-27)

```
### T3a: narrow grant (prune: images, build-cache) asks for volumes
not authorized to prune volumes: it is not in this tool's prune: list in mcp-sudo.yaml (granted: images, build-cache)
### T3c: no targets
targets is required: name what to reclaim (containers, images, volumes, networks, build-cache) - there is no implicit prune-everything
### T3d: typo
unknown prune target "image": use one or more of containers, images, volumes, networks, build-cache
```

### T5 - live prune, Docker-in-Docker host `fr011-dind` (2026-09-27)

Fixtures: 3 stopped containers, 2 dangling images, 1 anonymous volume, 1 named volume `fr012-named`, 1 unused network `fr012net`; `web-1`, `db-1`, `logger-1` running with `app-data`, `appnet`, `backend` in use.

```
$ linuxctl prune docker containers images networks
containers: 3 removed, 20.0 KiB reclaimed
  7d1dc187b4ff
  9a6c2228b515
  28fa44079c42
images: 2 removed, 18.8 KiB reclaimed
  untagged sha256:80e09c019f9d
  sha256:80e09c019f9d
  untagged sha256:e20b5d678151
  sha256:e20b5d678151
networks: 1 removed
  fr012net

Total reclaimed: 38.8 KiB

$ linuxctl prune docker volumes -o json
{
  "results": [
    {
      "target": "volumes",
      "deleted": [
        "c5724212278f01cb0b2e4bb36ebc051c2af0c9ba29c652b49062c2b5fbf880cb"
      ],
      "count": 1,
      "space_reclaimed_bytes": 0
    }
  ],
  "total_space_reclaimed_bytes": 0
}

=== survivors ===
db-1 [running]
web-1 [running]
logger-1 [running]
vol app-data
vol fr012-named
net appnet
net backend
net bridge
net host
net none
```

Raw MCP JSON-RPC over SSE (`http://localhost:9093`, user `privileged`):

```json
{
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
        "content": [
            {
                "type": "text",
                "text": "containers: 1 removed, 4.0 KiB reclaimed\n  97a565be7b77\n\nTotal reclaimed: 4.0 KiB\n"
            }
        ]
    }
}
```

### T6 - audit (2026-09-27)

```
2026-09-27T14:13:41.038Z INFO  tool call audit=true user=privileged session=b97121f1b9da120f-1 tool=docker/prune privileged=true duration_ms=10 ok=true args="{\"output_format\":\"json\",\"targets\":[\"containers\"]}" reclaimed="containers: 1 removed, 8192 bytes; total 8192 bytes"
2026-09-27T14:13:41.054Z INFO  tool call audit=true user=privileged session=022ff01f7a9bd142-2 tool=docker/prune privileged=true duration_ms=3 ok=true args="{\"output_format\":\"json\",\"targets\":[\"volumes\"]}" reclaimed="volumes: 1 removed, 0 bytes; total 0 bytes"
2026-09-27T14:13:41.070Z INFO  tool call audit=true user=privileged session=c26631d2a6ac3341-3 tool=docker/prune privileged=true duration_ms=2 ok=true args="{\"targets\":[\"containers\"]}" reclaimed="containers: 0 removed, 0.0 B reclaimed; Total reclaimed: 0.0 B"
```

### T7 - docs (2026-09-27)

```
$ bash scripts/check_docs.sh
== tools (46)
== resources (11)
== resource templates (10)
== stale grants
== reference example on the mcp-sudo page
✅ Every tool, resource and resource template has a docs page and an overview entry, every grant a privileged read needs is in the reference config, the mcp-sudo page shows that config as it is, and nothing listed is stale.

$ bash scripts/check_readmes.sh
Checking for missing README.md files...
✅ All packages have a README.md!
```

## Comments

- 2026-09-27 - created from FR-011 discussion. Deliberately out of scope for FR-011 because prune cannot be scoped by container name and needs its own `prune:` allowlist.
- 2026-09-27 - FR-013 (read-only Docker networks) is the *only* other place networks appear; `networks` as a prune target here is the only way this daemon ever removes one. Keep that split: FR-013 never writes, FR-012 is where network removal is granted.
- 2026-09-27 - `new/` → `in-progress/`: started after FR-011 and FR-013 reached `review/`, in the owner's order 11, 13, 12.
- 2026-09-27 - `filters` dropped from the proposed shape (see the scope note under Acceptance criteria). The final arguments are `targets` (required array) and `output_format`, nothing else.
- 2026-09-27 - three bugs found only by looking at live output, not by the unit tests: `untagged sha256:<64 hex>` printed unshortened (`short()` now recurses past the `untagged ` prefix); `images: 4 removed` for 2 images, because Docker reports one entry per *event* - untag then delete - so only entries with `Deleted != ""` are counted now; and `networks: 1 removed, 0.0 B reclaimed`, which read as a measurement the API never reports, so that clause is omitted for networks alone.
- 2026-09-27 - a fourth bug, found while capturing T6: `pruneSummary` assumed the text format, so an `-o json` call logged the whole JSON body - every deleted object's ID - into the audit line, breaking the "counts and bytes, never content" promise that justifies the exception to "tool output is never logged". It now summarizes the structured shape from its counts. Regression test: `TestPruneSummaryLogsCountsNotObjects` in `internal/rpc/docker_test.go`, which asserts neither shape leaks an object name.
- 2026-09-27 - `explain` listed this tool as `get docker prune`, because the resolver's `linuxctl_verb` doubles as a target keyword for read-shaped tools. Fixed at the shared point with a `verbShaped` set in `cmd/linuxctl/resolver.go`, which also corrected the same pre-existing wart for `docker/exec` (`get docker exec`).
- 2026-09-27 - all acceptance criteria met, T1-T7 pass, docs written with live output. Moving to `review/`.
