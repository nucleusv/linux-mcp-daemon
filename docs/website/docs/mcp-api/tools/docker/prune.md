# Prune

**Tool Name**: `docker/prune`

Reclaims disk by deleting unused Docker objects: stopped containers, dangling images, anonymous volumes, unused networks, and the build cache. `targets` is required and there is no "everything" - name each kind to reclaim. This is the only docker tool that deletes objects it was never given the names of, so it is scoped by kind rather than by container: which kinds this user may reclaim comes from the `prune:` list in their grant, not from the call, and a target outside that list refuses the whole request before anything is deleted. Two Engine API defaults are never overridden: images prunes only *dangling* ones (an image a stopped container still references survives), and volumes only *anonymous* ones (a named volume, the one thing in a Docker install nothing can rebuild, is never touched). Targets are reclaimed in dependency order - containers first, build cache last - whatever order they are given in. This tool always runs as root (the Docker socket is root-owned), so it takes no `privileged` argument - being able to call it at all means it was granted in mcp-sudo.yaml.

## What each target actually removes

| Target | Endpoint | Removes | Never removes | Reports space |
|---|---|---|---|---|
| `containers` | `POST /containers/prune` | stopped, exited, created containers | running, paused | yes |
| `images` | `POST /images/prune` | **dangling** images (untagged and unreferenced) | anything tagged, or referenced by any container | yes |
| `volumes` | `POST /volumes/prune` | **anonymous** unused volumes | **named** volumes, and any volume in use | yes |
| `networks` | `POST /networks/prune` | unused user-defined networks | `bridge`, `host`, `none`, or any with a container attached | no - the API reports none |
| `build-cache` | `POST /build/prune` | unused build cache records | in-use cache | yes |

The tool sends **no filters at all**, and that is the safety margin rather than an omission:

- `/images/prune` without filters removes only dangling images. `dangling=false` would remove every image no *running* container uses - most of a deployment host's cache.
- `/volumes/prune` without filters removes only anonymous volumes. `all=true` would take named ones.
- `/build/prune` without `all=true` removes only unused cache records.

Docker's `--force`, `--all` and `--filter until=…/label=…` are therefore not exposed. The knob is the `prune:` list in the grant, not an argument the caller picks.

:::warning Anonymous is not the same as worthless

A container started with `-v /var/lib/postgresql/data` (no name) gets an anonymous volume holding real data. Once that container is gone the volume is unused, and `volumes` will delete it. Named volumes are the ones that are structurally safe here.

And because one call prunes in dependency order, asking for `containers` **and** `volumes` together can delete an anonymous volume that was still attached when the call started - removing the container is what made it unused. Two separate calls, or leaving `volumes` out of the grant, avoids that.

:::

## Parameters

| Name | Type | Description |
|---|---|---|
| `targets` | array of string, **required** | One or more of `containers`, `images`, `volumes`, `networks`, `build-cache`. An empty or missing list is an error, never "everything". |
| `output_format` | string | `json`, `yaml`. Default is text. |

Counts are objects affected, not API events: Docker reports a removed image twice (untagged, then deleted) and both lines are listed, but only the deletion is counted. `networks` prints no reclaimed figure because the endpoint reports none - `0.0 B` would read as a measurement.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

The targets are positional words, since `targets` is the required array:

```bash
linuxctl prune docker containers images networks
```

Output (Docker-in-Docker test host: three stopped containers, two dangling images, one unused network, while `web-1`, `db-1` and `logger-1` keep running):
```text
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
```

```bash
linuxctl prune docker volumes -o json
```

Output - the anonymous volume goes, the named `fr012-named` and the in-use `app-data` stay:
```json
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
```

A target outside this user's `prune:` list refuses the whole call, before the socket is dialled - the granted targets in the same call are not reclaimed either:
```text
$ linuxctl prune docker volumes        # user granted prune: [images, build-cache]
not authorized to prune volumes: it is not in this tool's prune: list in mcp-sudo.yaml (granted: images, build-cache)

$ linuxctl prune docker               # no targets
targets is required: name what to reclaim (containers, images, volumes, networks, build-cache) - there is no implicit prune-everything

$ linuxctl prune docker image         # typo
unknown prune target "image": use one or more of containers, images, volumes, networks, build-cache
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
# 1. Open the SSE stream (in the background) and capture the one-time POST endpoint
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

# 2. POST the tools/call request to that endpoint
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "docker/prune", "arguments": {"targets": ["containers"]}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
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

</details>

## Permissions

Root or nothing, like every `docker/*` tool, plus its own allowlist of kinds - a `containers:` list cannot scope a tool that names no container:

```yaml
docker/prune:
  allowed: true
  prune:
    - images
    - build-cache
```

There is deliberately no `"*"` and no `all`. `allowed: true` with no `prune:` list is a strict-parse error (`linuxctl edit mcpd config sudo` refuses the save) and inert at startup - it refuses every call rather than allowing any, exactly like a missing `containers:` list. A misspelled target is rejected at load whatever the strictness, and `prune:` on any other tool is rejected as a no-op. See [Limiting what may be reclaimed](../../../configuration/mcp-sudo#limiting-what-may-be-reclaimed-prune).

Every call is audit-logged whatever the log level, with the requested target list and - uniquely for this tool - a one-line summary of what it reclaimed. That is the single exception to the daemon's rule that tool output is never logged: for a tool that deletes objects nobody named, what it deleted is the whole point of the audit line, and the summary carries counts and bytes, never content.

A partial failure still reports what was deleted (a per-target `error` alongside the counts); a call where every target failed is an error, not a report.

See [permissions and risks](../../../configuration/permissions-and-risks) for why any Docker socket grant is close to host root regardless.
