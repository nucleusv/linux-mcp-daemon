# DMI Hardware Info

**URI**: `devices://dmi`

Desktop Management Interface info (lshw/hwinfo equivalent). Detailed hardware specifications (RAM banks, BIOS, chassis).

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource devices://dmi
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "devices://dmi"}}'
```

Real response (captured live) - this test VM has no DMI table exposed (`/sys/class/dmi` is absent in this containerized environment), so this is a genuine failure, not a placeholder; on a normal host this returns real board/vendor data:
```json
{
  "jsonrpc": "2.0",
  "id": "d1",
  "error": {
    "code": -32603,
    "message": "worker execution failed: exit status 1. Stderr: DMI data not available on this system\n"
  }
}
```

</details>
