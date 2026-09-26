# Service Status

**URI Template**: `service://{name}/status`

Exposes DBus service properties (ActiveState, LoadState, SubState). Best used alongside the services/manage tool to check if a service actually started.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource service://ssh.service/status
```

Output (Ubuntu 24.04 VPS):
```json
{
  "name": "ssh.service",
  "description": "OpenBSD Secure Shell server",
  "load_state": "loaded",
  "active_state": "active",
  "sub_state": "running",
  "fragment_path": "/usr/lib/systemd/system/ssh.service"
}
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "service://ssh.service/status"}}'
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
        "text": "{\n  \"name\": \"ssh.service\",\n  \"description\": \"OpenBSD Secure Shell server\",\n  \"load_state\": \"loaded\",\n  \"active_state\": \"active\",\n  \"sub_state\": \"running\",\n  \"fragment_path\": \"/usr/lib/systemd/system/ssh.service\"\n}",
        "uri": "service://ssh.service/status"
      }
    ]
  }
}
```

</details>

