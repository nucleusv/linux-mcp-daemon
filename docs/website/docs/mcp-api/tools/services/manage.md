# Manage

**Tool Name**: `services/manage`

Starts, stops, restarts, reloads, enables or disables ONE systemd service over D-Bus (`.service` is appended when missing). Mutating. Ordinary users are usually refused by polkit, so `privileged: true` (needs a grant in mcp-sudo.yaml) is normally required. start, stop, restart and reload wait for the systemd job and return `Job N completed with status: done` (or failed, canceled, timeout, dependency, skipped); a slow one can hit the 30 s worker limit. `reload` asks the service to re-read its config without stopping it, only if the unit supports it. `enable` and `disable` only change whether it starts at boot; they do not start or stop it. To see state use `services/list` or the `service://<name>/status` resource, for logs `logs/journal-control`, for containers `docker/manage`, for a raw signal to a PID `processes/delete`.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl restart system services nginx.service --privileged true
```

Output (not executed here - hypothetical, matches the tool's source code):
```text
Job 482 completed with status: done
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "services/manage", "arguments": {"service": "nginx.service", "action": "restart", "privileged": true}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response (not executed here - hypothetical, matches the tool's source code):
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Job 482 completed with status: done"
      }
    ]
  }
}
```

</details>

