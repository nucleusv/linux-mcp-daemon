# Logins

**Tool Name**: `logs/logins`

Lists login history (wraps `last`) or failed login attempts (`type: "failed"`, wraps `lastb`). Returns raw text, not JSON - `last`/`lastb`'s output isn't safe to hand-parse into structured data reliably.

`type: "failed"` typically requires `privileged: true`, since `btmp` is usually root-only readable. When `mcpd` runs containerized, `privileged: true` also automatically reads the real host's login history - see [Master Daemon Configuration](../../../configuration/daemon.md)'s `worker.containerized` setting.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get logs logins --privileged true
```

Output:
```text
last failed: exec: "last": executable file not found in $PATH
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "logs/logins", "arguments": {"privileged": true}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "23",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "last failed: exec: \"last\": executable file not found in $PATH"
      }
    ],
    "isError": true
  }
}
```

</details>

