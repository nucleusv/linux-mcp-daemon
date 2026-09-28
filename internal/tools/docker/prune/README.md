# docker/prune

Reclaims disk by deleting unused Docker objects over the Engine API: `POST /containers/prune`, `/images/prune`, `/volumes/prune`, `/networks/prune`, `/build/prune`.

The only tool in this daemon that deletes objects it was never given the names of. `docker/manage remove` takes one container and is scoped by a `containers:` list; prune is scoped by *what kind of garbage* to collect, so it carries its own allowlist.

## Parameters
- `target` (string, **required**): exactly one of `containers`, `images`, `volumes`, `networks`, `build-cache`. There is no `all` — a missing target is an error, never "everything".
- `output_format` (string, optional): `json`, `yaml`; default text.

One kind per call, deliberately. The kinds are not independent — removing stopped containers is what makes their images dangling and their anonymous volumes unused — so a call naming two of them reclaims more than the sum of what either name described, and nothing in the request says which of those deletions was actually asked for. Separate calls make each deletion its own decision, and its own audit line. Call it again for the next kind; the effects still compound across calls, just visibly.

## What each target actually removes

| Target | Endpoint | Removes | Reports space |
|---|---|---|---|
| `containers` | `/containers/prune` | stopped containers | yes |
| `images` | `/images/prune` | **dangling** images only | yes |
| `volumes` | `/volumes/prune` | **anonymous**, unused volumes only | yes |
| `networks` | `/networks/prune` | unused user-defined networks | no (the API reports none) |
| `build-cache` | `/build/prune` | the builder's cache | yes |

Two Engine API defaults are the safety margin here and this tool never overrides them — it sends no filters at all:

- `/images/prune` without filters removes only dangling (untagged, unreferenced) images. `dangling=false` would remove every image no *running* container uses, i.e. most of a deployment host's cache.
- `/volumes/prune` without filters removes only anonymous volumes. `all=true` would take named ones, and a named volume holds the one thing in a Docker install nothing can rebuild.

## Usage & Permissions
⚠ Root or nothing, plus its own allowlist:

```yaml
docker/prune:
  allowed: true
  prune:
    - images
    - build-cache
```

`allowed: true` with no `prune:` list is a strict-parse error (`linuxctl edit mcpd config sudo` refuses the save) and inert at startup — it refuses every call rather than allowing any. A target outside the list is refused before the socket is dialled, so an unauthorized request deletes nothing at all. A misspelled target in the config is rejected at load, strict or not.

Every call is audit-logged whatever the log level, with the target and — uniquely for this tool — a one-line summary of what it reclaimed, because for a destructive tool that is the whole point of the audit line. One kind per call is also what makes that line unambiguous.

A prune that fails is an error, not a report of nothing.

`linuxctl prune docker images` calls this tool — the target is a positional word.
