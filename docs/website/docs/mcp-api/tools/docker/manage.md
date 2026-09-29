# Manage

**Tool Name**: `docker/manage`

Container lifecycle: start, stop, restart, kill, pause, unpause, remove - what services/manage is for systemd units. Which containers it may touch comes from the `containers:` list in this user's grant, not from the call. `remove` is the one irreversible action and is deliberately fenced: it never sends Docker's `force` or `v`, so a running container is refused (stop or kill it first, two separate audited calls) and anonymous volumes are never deleted. Restarting a container that holds state, or removing one that is not reproducible from its image and volumes, loses work - check container://\<name\>/status first. This tool always runs as root (the Docker socket is root-owned), so it takes no `privileged` argument - a call is refused unless this user's grant in mcp-sudo.yaml allows it.

The `containers:` list in the grant is the whole authorization boundary - it is matched in the worker, against both the name and the resolved ID, before anything is sent to the socket. A grant with no `containers:` list refuses everything.

:::warning `containers: ["*"]` includes mcpd's own container

When `mcpd` itself runs as a container on the same daemon, `["*"]` matches it too - so a granted user can `stop` or `remove` the very container serving the call. The stop succeeds and the answer never arrives. List the containers the user is meant to manage, or a glob that cannot match mcpd's own name. The same caution applies to the Docker daemon's own infrastructure containers. See [Permissions and risks](../../../configuration/permissions-and-risks.md).

:::

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

Every action is its own verb - the action comes from the enum in the schema, so `start`, `stop`, `restart`, `kill`, `pause`, `unpause` and `remove` all work the same way:

```bash
linuxctl restart docker container web-1
```

Output (Docker-in-Docker test host):
```text
Container web-1 (e36aeba4369a): restart succeeded.
```

```bash
linuxctl start docker container db-1 -o json
```

Output:
```json
{
  "action": "start",
  "container": "db-1",
  "id": "80d3cf9c8b3c4f5ed7b5a9d52c9f97d9759f585d0e9214f54646725eda7c0363",
  "result": "ok"
}
```

An action the container is already in is a no-op, not a failure - Docker answers `304` and the tool reports it as `unchanged` rather than an error an agent would retry:

```bash
linuxctl start docker container db-1
```

Output (`db-1` already running):
```text
Container db-1 (80d3cf9c8b3c): start not needed, it is already in that state.
```

`remove` never forces. A running container is refused, and the error says what to do about it:

```bash
linuxctl remove docker container web-1
```

Output:
```text
docker API returned 409: cannot remove container "e36aeba4369a725068b13142ef53c541abdfbaf857cfdc4b8370210e095a2099": container is running: stop the container before removing or force remove - stop or kill it first (this tool never removes a running container by force)
```

A container outside the caller's `containers:` list is refused before the socket is dialled - here a user granted `containers: ["web-*"]` reaching for `db-1`:

```bash
linuxctl stop docker container db-1
```

Output:
```text
not authorized to act on container db-1 (80d3cf9c8b3c): it is not in this tool's containers: list in mcp-sudo.yaml
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "docker/manage", "arguments": {"container": "web-1", "action": "restart"}}}'

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
        "text": "Container web-1 (e36aeba4369a): restart succeeded.\n"
      }
    ]
  }
}
```

</details>
