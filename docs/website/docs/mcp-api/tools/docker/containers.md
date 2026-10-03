# Containers

**Tool Name**: `docker/containers`

Lists Docker containers through the Engine API on the local socket (no `docker` CLI needed): name, image, state, status, ports. Read-only. Only running containers unless `all: true`; `state` (created, running, paused, exited, ...) filters to one state and implies `all`; `pattern` is a glob on the container name (`web-*`); `limit` returns the newest N. Every container is listed: the grant's `containers:` list applies only to docker/manage, docker/logs and docker/exec. Text is a block per container plus a hint line, or `No containers found matching the criteria.`; `output_format: json` (also yaml/table/wide) returns an array of objects (name, names, id, full_id, image, image_id, command, state, status, created, ports, labels), `[]` when empty. For logs use `docker/logs`, to start or stop `docker/manage`, for one container's details the `docker-container://<name>/status` resource. Always runs as root: no `privileged` argument, refused unless the user's grant has `allowed: true`.

Every `docker/*` tool needs `allowed: true` in its own grant in [mcp-sudo.yaml](../../../configuration/mcp-sudo.md) - without the grant the tool is still listed, but calling it fails with an error naming the missing grant, the way `daemon/reload-config` does. `docker/containers` is a plain listing, so it has no `containers:` allowlist: a user who may list at all sees every container on the host. When `mcpd` runs containerized, point `worker.docker_socket` in [daemon.yaml](../../../configuration/daemon.md) at the socket as the worker sees it.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get docker containers
```

Output (Docker-in-Docker test host):
```text
[running] web-1 (e36aeba4369a)
  Image: nginx:alpine
  Status: Up 5 minutes
  Ports: 0.0.0.0:8080->80/tcp, :::8080->80/tcp

[running] logger-1 (b2171b5ad4e6)
  Image: busybox
  Status: Up 6 hours

Hint: For one container's runtime summary, read docker-container://<name>/status; for the full config, docker-container://<name>/inspect.
```

A stopped container only shows with `--all`:

```bash
linuxctl get docker containers --all
```

Output:
```text
[exited] db-1 (80d3cf9c8b3c)
  Image: redis:alpine
  Status: Exited (0) Less than a second ago

[running] web-1 (e36aeba4369a)
  Image: nginx:alpine
  Status: Up 5 minutes
  Ports: 0.0.0.0:8080->80/tcp, :::8080->80/tcp

[running] logger-1 (b2171b5ad4e6)
  Image: busybox
  Status: Up 6 hours

Hint: For one container's runtime summary, read docker-container://<name>/status; for the full config, docker-container://<name>/inspect.
```

`-o json` adds the fields the text view leaves out - full ID, image ID, creation time, the entrypoint command and labels:

```bash
linuxctl get docker containers --state running -o json
```

Output (one element of the array):
```json
{
  "command": "/docker-entrypoint.sh nginx -g 'daemon off;'",
  "created": "2026-09-27T10:04:49Z",
  "full_id": "e36aeba4369a725068b13142ef53c541abdfbaf857cfdc4b8370210e095a2099",
  "id": "e36aeba4369a",
  "image": "nginx:alpine",
  "image_id": "sha256:df221db836e1754089190208cee7eeda94f233197056426eda74a43ab1abeac2",
  "labels": {
    "maintainer": "NGINX Docker Maintainers <docker-maint@nginx.com>"
  },
  "name": "web-1",
  "names": [
    "web-1"
  ],
  "ports": "0.0.0.0:8080->80/tcp, :::8080->80/tcp",
  "state": "running",
  "status": "Up 4 minutes"
}
```

`name` is the single name a caller would use; `names` is Docker's own array, which has more than one element when a container carries network aliases.

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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "docker/containers", "arguments": {"all": true}}}'

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
        "text": "[exited] db-1 (80d3cf9c8b3c)\n  Image: redis:alpine\n  Status: Exited (0) Less than a second ago\n\n[running] web-1 (e36aeba4369a)\n  Image: nginx:alpine\n  Status: Up 5 minutes\n  Ports: 0.0.0.0:8080->80/tcp, :::8080->80/tcp\n\n[running] logger-1 (b2171b5ad4e6)\n  Image: busybox\n  Status: Up 6 hours\n\nHint: For one container's runtime summary, read docker-container://<name>/status; for the full config, docker-container://<name>/inspect.\n"
      }
    ]
  }
}
```

</details>
