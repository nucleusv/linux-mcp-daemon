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

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "disks://vda/stats"}}'
```

</details>

**Real response (captured live):**

```json
[
  {
    "major": 254,
    "minor": 0,
    "device_name": "vda",
    "reads_completed": 9558738,
    "reads_merged": 506656,
    "sectors_read": 4052204802,
    "time_reading_ms": 9788136,
    "writes_completed": 2162223,
    "writes_merged": 1689822,
    "sectors_written": 78537752,
    "time_writing_ms": 5850628,
    "ios_in_progress": 0,
    "time_doing_ios_ms": 1742727,
    "weighted_time_ios_ms": 16081971
  }
]
```
