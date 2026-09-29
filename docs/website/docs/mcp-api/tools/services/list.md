# List

**Tool Name**: `services/list`

Lists systemd `.service` units that systemd currently has loaded, with load, active and sub state (D-Bus ListUnits). Read-only. An installed but never-loaded unit file may be missing, and timers (use [`timers/list`](../timers/list)), sockets and other unit types are not included. `pattern` supports only a leading and/or trailing `*` (`kube*`, `*ssh*`); without `*` it is an exact unit name including `.service`. The state filters are exact strings: `active_state` active/failed/inactive, `sub_state` running/exited/dead, `load_state` loaded/not-found. Text output is a block per unit plus a hint line, or `No services found matching the criteria.`; `output_format: json` returns an array of objects (name, description, load_state, active_state, sub_state), `[]` when empty. To change a service use `services/manage`, for one unit's details the `service://<name>/status` resource, for its logs `logs/journal-control`.

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
linuxctl get system services --pattern "ssh*"
```

Output (Ubuntu 24.04 VPS):
```text
[active] ssh.service
  State: running (active) | Load: loaded
  Desc: OpenBSD Secure Shell server

Hint: To read detailed properties of a specific service, use the resource: service://<name>/status
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "services/list", "arguments": {"pattern": "ssh*"}}}'

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
        "text": "[active] ssh.service\n  State: running (active) | Load: loaded\n  Desc: OpenBSD Secure Shell server\n\nHint: To read detailed properties of a specific service, use the resource: service://<name>/status\n"
      }
    ]
  }
}
```

</details>

