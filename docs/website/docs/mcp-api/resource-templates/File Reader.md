# File Reader

**URI Template**: `file:///{path}`

Reads any file on the system. Append /stat for file metadata, /content for contents, or /type for file type.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource file:///etc/hosts/stat
```

Output:
```text
Error: Permission denied. Hint: You are not authorized to use 'privileged: true' for this tool in mcp-sudo.yaml (Code: -32603)
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "file:///etc/hosts/stat"}}'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "41",
  "error": {
    "code": -32603,
    "message": "Permission denied. Hint: You are not authorized to use 'privileged: true' for this tool in mcp-sudo.yaml"
  }
}
```

</details>

