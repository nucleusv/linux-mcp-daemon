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

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "file:///etc/hosts/stat"}}'
```

</details>

**Real response (captured live) - this particular URI needs `privileged: true` under the hood, and the token used for this capture wasn't granted that in `mcp-sudo.yaml`, so this is a genuine authorization error, not a placeholder:**

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
