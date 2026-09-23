# List

**Tool Name**: `services/list`

Lists systemd services with optional filtering by name pattern, active state, load state, and sub state. Output includes `ActiveState`, `LoadState`, and `SubState` for each matching `.service` unit. To get detailed service properties and state for a single service, read the `service://{name}/status` resource. To control a service, use the `services/manage` tool.

## Arguments

| Argument | Type | Description |
| --- | --- | --- |
| `pattern` | string | Wildcard pattern to match service names (e.g., `kube*`, `*ssh*`) |
| `active_state` | string | Filter by active state (e.g., `active`, `failed`, `inactive`) |
| `load_state` | string | Filter by load state (e.g., `loaded`, `not-found`) |
| `sub_state` | string | Filter by sub state (e.g., `running`, `exited`, `dead`) |
| `output_format` | string | Desired output format (e.g. `json`, `table`, `wide`). Defaults to text |
| `privileged` | boolean | Run as root (may be required depending on policies) |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get system services --pattern "kube*"
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
# 1. Open the SSE stream (in the background) and capture the one-time POST endpoint
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

# 2. POST the tools/call request to that endpoint
curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "services/list", "arguments": {"pattern": "kube*"}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Real response (captured live):
```text
[active] kubelet.service
  State: running (active) | Load: loaded
  Desc: kubelet: The Kubernetes Node Agent

Hint: To read detailed properties of a specific service, use the resource: service://<name>/status
```

</details>
