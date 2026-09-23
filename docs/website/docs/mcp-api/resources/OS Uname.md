# OS Uname

**URI**: `os://uname`

Native system uname information (kernel version, node name). Hint: For CPU hardware architecture use cpu/list tool.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource os://uname
```

Output:
```text
Sysname: Linux
Nodename: desktop-control-plane
Release: 7.0.12-linuxkit
Version: #1 SMP PREEMPT Thu Aug 27 14:02:21 UTC 2026
Machine: aarch64
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "os://uname"}}'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "31",
  "result": {
    "contents": [
      {
        "mimeType": "text/plain",
        "text": "Sysname: Linux\nNodename: desktop-control-plane\nRelease: 7.0.12-linuxkit\nVersion: #1 SMP PREEMPT Thu Aug 27 14:02:21 UTC 2026\nMachine: aarch64",
        "uri": "os://uname"
      }
    ]
  }
}
```

</details>

