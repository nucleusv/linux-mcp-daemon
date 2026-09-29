# Performance

**Tool Name**: `disks/performance`

Returns block-device I/O counters from /proc/diskstats: reads and writes completed and merged, sectors (512 bytes) and milliseconds spent, in-flight I/Os and weighted I/O time. Values are CUMULATIVE since boot, not rates and not iostat's per-interval figures; there is no %util or await, so sample twice and subtract to get a rate. Read-only. Without `device` all devices are listed except `loop*` and `ram*`; a name such as `sda` (find them with `disks/list`) selects one, and an unknown name gives `no such block device`. Text output is a table; `output_format: json` and `yaml` (real YAML here) return an array of objects with 14 fields (major, minor, device_name, reads_completed, ..., weighted_time_ios_ms). For capacity use `disks/free`, for SMART health `disks/health`.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get disks performance vda
```

Output (Ubuntu 24.04 VPS):
```text
Device       Reads        Writes       SectRead     SectWrite    I/O(ms)     
----------------------------------------------------------------------------
vda          18284        208946       1869176      5710280      26466       
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "disks/performance", "arguments": {"device": "vda"}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Device       Reads        Writes       SectRead     SectWrite    I/O(ms)     \n----------------------------------------------------------------------------\nvda          18284        208946       1869176      5710280      26466       \n"
      }
    ]
  }
}
```

</details>

