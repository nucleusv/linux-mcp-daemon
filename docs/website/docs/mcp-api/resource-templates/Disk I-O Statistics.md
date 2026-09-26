# Disk I/O Statistics

**URI Template**: `disks://{name}/stats`

Real-time I/O statistics for a specific block device (e.g. sda). Returns JSON.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource disks://vda/stats
```

Output (Ubuntu 24.04 VPS):
```json
[
  {
    "major": 253,
    "minor": 0,
    "device_name": "vda",
    "reads_completed": 18279,
    "reads_merged": 4481,
    "sectors_read": 1868616,
    "time_reading_ms": 7478,
    "writes_completed": 208573,
    "writes_merged": 102639,
    "sectors_written": 5705840,
    "time_writing_ms": 102022,
    "ios_in_progress": 0,
    "time_doing_ios_ms": 26452,
    "weighted_time_ios_ms": 118412
  }
]
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "disks://vda/stats"}}'
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
        "text": "[\n  {\n    \"major\": 253,\n    \"minor\": 0,\n    \"device_name\": \"vda\",\n    \"reads_completed\": 18279,\n    \"reads_merged\": 4481,\n    \"sectors_read\": 1868616,\n    \"time_reading_ms\": 7478,\n    \"writes_completed\": 208573,\n    \"writes_merged\": 102639,\n    \"sectors_written\": 5705840,\n    \"time_writing_ms\": 102022,\n    \"ios_in_progress\": 0,\n    \"time_doing_ios_ms\": 26452,\n    \"weighted_time_ios_ms\": 118412\n  }\n]",
        "uri": "disks://vda/stats"
      }
    ]
  }
}
```

</details>

