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

Output (Ubuntu 24.04 VPS):
```text
Sysname: Linux
Nodename: vps.example.com
Release: 6.8.0-142-generic
Version: #142-Ubuntu SMP PREEMPT_DYNAMIC Wed Sep  2 14:24:27 UTC 2026
Machine: x86_64
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "os://uname"}}'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "contents": [
      {
        "mimeType": "text/plain",
        "text": "Sysname: Linux\nNodename: vps.example.com\nRelease: 6.8.0-142-generic\nVersion: #142-Ubuntu SMP PREEMPT_DYNAMIC Wed Sep  2 14:24:27 UTC 2026\nMachine: x86_64",
        "uri": "os://uname"
      }
    ]
  }
}
```

</details>

