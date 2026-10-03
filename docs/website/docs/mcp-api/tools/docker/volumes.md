# Volumes

**Tool Name**: `docker/volumes`

Lists Docker volumes with driver, mountpoint (a host path) and the names of the containers, running or stopped, that mount each. Read-only: volumes are never created or removed here. `pattern` is a glob on the volume name. Text is a block per volume; `output_format: json` (also yaml/table/wide) returns an array of objects (name, driver, mountpoint, created, scope, labels, options, in_use_by), `[]` when empty. `docker/prune` with target `volumes` removes anonymous unused volumes only. For one volume's details use the `docker-volume://<name>/inspect` resource, for containers `docker/containers`. Always runs as root: no `privileged` argument, refused unless the grant has `allowed: true`.

`In use by` is computed from the container list, so it names containers whether they are running or stopped - a stopped container still holds its volume. An empty list means no container mounts it, which is the closest thing to "safe to delete" this tool will tell you; deleting it is still someone else's job.

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `output_format` | string | no | Use json for structured output (yaml, table and wide return the same JSON); default is text |
| `pattern` | string | no | Only volumes whose name matches this glob |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get docker volumes
```

Output (Docker-in-Docker test host):
```text
app-data
  Driver: local | Scope: local
  Mountpoint: /var/lib/docker/volumes/app-data/_data
  In use by: web-1

Hint: For one volume's options and labels, read docker-volume://<name>/inspect
```

```bash
linuxctl get docker volumes -o json
```

Output:
```json
[
  {
    "created": "2026-09-27T10:04:49Z",
    "driver": "local",
    "in_use_by": [
      "web-1"
    ],
    "labels": null,
    "mountpoint": "/var/lib/docker/volumes/app-data/_data",
    "name": "app-data",
    "options": null,
    "scope": "local"
  }
]
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "docker/volumes", "arguments": {}}}'

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
        "text": "app-data\n  Driver: local | Scope: local\n  Mountpoint: /var/lib/docker/volumes/app-data/_data\n  In use by: web-1\n\nHint: For one volume's options and labels, read docker-volume://<name>/inspect\n"
      }
    ]
  }
}
```

</details>
