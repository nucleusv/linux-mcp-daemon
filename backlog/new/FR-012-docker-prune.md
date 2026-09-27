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

- [ ] `docker/prune` tool implemented against the Docker Engine API (`/containers/prune`, `/images/prune`, `/volumes/prune`, `/networks/prune`, `/build/prune`).
- [ ] Requires `privileged: true`; no unprivileged fallback.
- [ ] `prune:` allowlist enforced; strict config load rejects `allowed: true` without a non-empty `prune:` list.
- [ ] Each target returns the count and reclaimed space (where the API provides it) or a clear message.
- [ ] Audit-logged with the full target list and reclaimed summary.
- [ ] `configs/mcp-sudo.yaml` reference `privileged` block updated with `docker/prune`.
- [ ] Docs page and overview list updated.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Prune images | Go unit test against fake Docker API | hits `/images/prune`, returns count/reclaimed | | |
| T2 | Multiple targets | Go unit test | hits each requested endpoint | | |
| T3 | Not authorized target | Grant `prune: ["containers"]`, call with `targets: ["volumes"]` | refused before any API call | | |
| T4 | Missing prune list | `ParseSudoConfig` strict | load error | | |
| T5 | Live: prune stopped containers | VPS with throwaway stopped container | container removed, count reported | | |
| T6 | Audit | `journalctl -u mcpd` | `docker/prune` call logged with targets | | |
| T7 | Docs | `check_docs.sh` | passes | | |

## Comments

- 2026-09-27 - created from FR-011 discussion. Deliberately out of scope for FR-011 because prune cannot be scoped by container name and needs its own `prune:` allowlist.
- 2026-09-27 - FR-013 (read-only Docker networks) is the *only* other place networks appear; `networks` as a prune target here is the only way this daemon ever removes one. Keep that split: FR-013 never writes, FR-012 is where network removal is granted.
