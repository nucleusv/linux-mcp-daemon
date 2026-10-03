# Manage

**Tool Name**: `docker/manage`

Changes a container's lifecycle: start, stop, restart, kill, pause, unpause or remove; the container counterpart of `services/manage`. Mutating. Only containers in the grant's `containers:` list may be touched (name and ID are both checked; an empty list refuses everything). Always runs as root: no `privileged` argument, refused unless the grant has `allowed: true`. `stop` uses Docker's default 10 s grace. `remove` is irreversible for the container and deliberately fenced: it never sends `force` or volume removal, so a running container is refused (`stop or kill it first`) and anonymous volumes are kept. `start` on a running or `stop` on a stopped container reports `unchanged`, not an error. `container` is a name, full ID or ID prefix. Text: `Container NAME (ID12): stop succeeded.`; `output_format: json` returns container, id, action and result (`ok` or `unchanged`). For bulk cleanup use `docker/prune`, to run a command inside `docker/exec`, to check state first `docker/containers`.

The `containers:` list in the grant is the whole authorization boundary - it is matched in the worker, against both the name and the resolved ID, before anything is sent to the socket. A grant with no `containers:` list refuses everything.

:::warning `containers: ["*"]` includes mcpd's own container

When `mcpd` itself runs as a container on the same daemon, `["*"]` matches it too - so a granted user can `stop` or `remove` the very container serving the call. The stop succeeds and the answer never arrives. List the containers the user is meant to manage, or a glob that cannot match mcpd's own name. The same caution applies to the Docker daemon's own infrastructure containers. See [Permissions and risks](../../../configuration/permissions-and-risks.md).

:::

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `action` | string | yes | Action to perform. One of: `start`, `stop`, `restart`, `kill`, `pause`, `unpause`, `remove`. |
| `container` | string | yes | Container name, full ID or ID prefix |
| `output_format` | string | no | json returns container, id, action, result; default is text |

## Actions

`action` is one of seven words. Every one acts on the container's resolved ID (the one that was authorized), so a concurrent `docker rename` cannot point it at another container. In `linuxctl` each action is its own verb: `linuxctl <action> docker container <name>`.

| Action | `linuxctl` | What it does | Reversible? | Already in that state |
| --- | --- | --- | --- | --- |
| `start` | `linuxctl start docker container web-1` | Starts a stopped or created container, like `docker start`. | Yes (`stop`) | Running container: `unchanged`, not an error |
| `stop` | `linuxctl stop docker container web-1` | Asks the main process to exit (`SIGTERM`); after Docker's default **10 s** grace period it is killed (`SIGKILL`). Like `docker stop`. | Yes (`start`) | Stopped container: `unchanged` |
| `restart` | `linuxctl restart docker container web-1` | `stop` (same 10 s grace) followed by `start`. A stopped container is just started. Like `docker restart`. | n/a | Always runs |
| `kill` | `linuxctl kill docker container web-1` | Ends the main process at once with `SIGKILL`, with no grace period. The container stops but still exists. Like `docker kill` with no signal argument; a different signal cannot be chosen. | Yes (`start`), but the process loses any unsaved state | Stopped container: Docker refuses (the container is not running) |
| `pause` | `linuxctl pause docker container web-1` | Freezes every process in the container with the cgroup freezer. Memory and state are kept; nothing runs and nothing is signalled. Like `docker pause`. | Yes (`unpause`) | Already paused: refused by Docker |
| `unpause` | `linuxctl unpause docker container web-1` | Resumes a paused container exactly where it stopped. Like `docker unpause`. | n/a | Not paused: refused by Docker |
| `remove` | `linuxctl remove docker container web-1` | Deletes the container. **Irreversible for the container**, and fenced: it never sends Docker's `force` nor volume removal. A *running* container is refused (`stop or kill it first`), and anonymous volumes stay behind (clean those with [`docker/prune`](./prune)). Like `docker rm`. | **No** | Missing container: Docker's 404 |

How the result is reported:
- `Container NAME (ID12): ACTION succeeded.` on success; with `output_format: json` you get `{action, container, id, result}` where `result` is `ok` or `unchanged`.
- `unchanged` (Docker answered `304`) is deliberately not an error: an agent would otherwise retry a `start` that has nothing to do.
- Anything else from Docker comes back as `docker API returned <status>: <Docker's own message>`.

Typical orderings: `stop` then `remove` to delete a running container; `kill` instead of `stop` only when a process ignores `SIGTERM`; `pause`/`unpause` to freeze a workload without losing its memory. To check the state first use [`docker/containers`](./containers); to run a command inside use [`docker/exec`](./exec).

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
