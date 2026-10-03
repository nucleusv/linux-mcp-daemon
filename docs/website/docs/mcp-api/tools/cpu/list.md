# List

**Tool Name**: `cpu/list`

Lists the machine's CPUs from /proc/cpuinfo. Read-only. Text output shows the number of logical processors and, for the FIRST processor only, vendor, model name, MHz (or BogoMIPS) and cache size. `output_format: json` (also yaml/table/wide) returns an array with one object per logical CPU using /proc/cpuinfo's own field names, which differ by architecture (x86 `model name`, `cpu MHz`, `flags`; ARM `CPU implementer`). It does not report sockets, cores or threads separately, and `topology_only` has no effect. For current load use `cpu/load-average`, for per-process CPU `processes/top`, for OS and kernel `system/os-release`.

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `output_format` | string | no | Use json for structured output (yaml, table and wide return the same JSON); default is text |
| `topology_only` | boolean | no | Has no effect: accepted but ignored, the output is the same |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get cpu
```

Output:
```text
CPU Information (Total Processors: 4)
Vendor ID: 0x61
Model Name: 8
CPU MHz/BogoMIPS: 48.00
Cache Size: N/A
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "cpu/list", "arguments": {}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "25",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "CPU Information (Total Processors: 4)\nVendor ID: 0x61\nModel Name: 8\nCPU MHz/BogoMIPS: 48.00\nCache Size: N/A"
      }
    ]
  }
}
```

</details>

