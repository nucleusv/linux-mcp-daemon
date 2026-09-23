# List

**Tool Name**: `files/list`

Lists contents of a directory.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get files list /var/log
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "files/list", "arguments": {"path": "/var/log"}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Real response (captured live):
```text
[FILE] alternatives.log (6522 bytes, modified: 2026-09-22 23:51:44)
[DIR]  apt/ (modified: 2026-09-22 23:51:42)
[FILE] bootstrap.log (61237 bytes, modified: 2026-09-11 02:06:15)
[FILE] btmp (0 bytes, modified: 2026-09-11 02:06:06)
[FILE] dpkg.log (215150 bytes, modified: 2026-09-22 23:51:45)
[FILE] faillog (0 bytes, modified: 2026-09-11 02:06:14)
[FILE] lastlog (0 bytes, modified: 2026-09-11 02:06:06)
[FILE] wtmp (0 bytes, modified: 2026-09-11 02:06:06)
```

</details>
