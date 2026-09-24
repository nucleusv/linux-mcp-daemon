# Usage

**Tool Name**: `memory/usage`

Returns memory and swap utilization information. Use cpu/load-average to check compute load.

| Parameter | Description |
|---|---|
| `human_readable` | Sizes like `free -h` (`1.8Gi`, `300Mi`); default is bytes, like `free -b` |
| `detailed` | Return the raw `/proc/meminfo` instead of the summary |
| `output_format` | `json`/`yaml`/`table`/`wide`: `total`, `used`, `free`, `shared`, `buffCache`, `available`, `swap_total`, `swap_used`, `swap_free` - always in bytes |

Sizes are in **bytes** by default, laid out like `free -b` with a `Swap` row; `human_readable: true` gives `free -h`. Structured output (`json`/`yaml`) is always in bytes.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get memory usage
```

Output (Ubuntu 24.04 VPS, 2 GB RAM):
```text
              total         used         free       shared   buff/cache    available
Mem:     1984503808    346693632    366071808      4481024   1271738368   1432211456
Swap:    3955224576            0   3955224576
```

</details>

<details>
<summary><b>linuxctl (human_readable)</b></summary>

```bash
linuxctl get memory usage --human_readable true
```

Output:
```text
              total         used         free       shared   buff/cache    available
Mem:          1.8Gi        323Mi        357Mi        4.3Mi        1.2Gi        1.3Gi
Swap:         3.7Gi           0B        3.7Gi
```

</details>

<details>
<summary><b>linuxctl (JSON)</b></summary>

```bash
linuxctl get memory usage -o json
```

Output:
```json
{
  "available": 1450258432,
  "buffCache": 1271738368,
  "free": 384110592,
  "shared": 4481024,
  "swap_free": 3955224576,
  "swap_total": 3955224576,
  "swap_used": 0,
  "total": 1984503808,
  "used": 328654848
}
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "memory/usage", "arguments": {}}}'

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
        "text": "              total         used         free       shared   buff/cache    available\nMem:     1984503808    346693632    366071808      4481024   1271738368   1432211456\nSwap:    3955224576            0   3955224576\n"
      }
    ]
  }
}
```

</details>
