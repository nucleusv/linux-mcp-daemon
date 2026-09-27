# Docker Volume Inspect

**URI Template**: `volume://{name}/inspect`

Driver, mountpoint, options and labels of one Docker volume. Hint: list volumes - with the containers mounting each - using the docker/volumes tool.

A read needs two grants in [mcp-sudo.yaml](../../configuration/mcp-sudo.md): the template itself (`volume://{name}/inspect`) and the internal worker it spawns (`docker/inspect`). A volume name is a single identifier, so a slash in it is refused rather than cleaned - the name can never address another Engine API endpoint.

Which container mounts a volume is not in this answer; Docker keeps only a refcount. The `docker/volumes` tool cross-references the container list and reports the names.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource volume://app-data/inspect
```

Output (Docker-in-Docker test host):
```json
{
  "CreatedAt": "2026-09-27T10:04:49Z",
  "Driver": "local",
  "Labels": null,
  "Mountpoint": "/var/lib/docker/volumes/app-data/_data",
  "Name": "app-data",
  "Options": null,
  "Scope": "local"
}
```

`Mountpoint` is a path on the Docker host, not inside any container - and for a `local` volume it is readable with the `files/*` tools, which is the usual way to look at what a volume holds without an exec into something that mounts it. A volume created with driver options (an NFS or CIFS mount) reports them in `Options`.

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "volume://app-data/inspect"}}'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "contents": [
      {
        "mimeType": "application/json",
        "text": "{\n  \"CreatedAt\": \"2026-09-27T10:04:49Z\",\n  \"Driver\": \"local\",\n  \"Labels\": null,\n  \"Mountpoint\": \"/var/lib/docker/volumes/app-data/_data\",\n  \"Name\": \"app-data\",\n  \"Options\": null,\n  \"Scope\": \"local\"\n}",
        "uri": "volume://app-data/inspect"
      }
    ]
  }
}
```

</details>
