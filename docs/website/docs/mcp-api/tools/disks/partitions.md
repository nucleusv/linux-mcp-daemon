# Partitions

**Tool Name**: `disks/partitions`

Retrieves partition boundaries for a block device (conceptually `fdisk -l`), parsed natively from `/sys/class/block` - this daemon does not wrap the `fdisk` binary. Returns each partition's device name, parent disk, partition number, and start/size in both sectors and bytes. No root required.

## Parameters

- `device` (optional) - only return partitions belonging to this disk (e.g. `vda`). Omit to list partitions across every disk on the system.
- `output_format` (optional) - `json` for structured output; omit for a plain text table.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
$ linuxctl get disks partitions vda --output json
[
  {
    "device": "vda1",
    "parent_disk": "vda",
    "number": 1,
    "start_sector": 2048,
    "size_sectors": 124997632,
    "size_bytes": 63998787584
  }
]
```

Output:
```text
DEVICE         PARENT     NUM    START(SECT)    SIZE(SECT)     SIZE(BYTES)   
vda1           vda        1      2048           124997632      63998787584
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "disks/partitions", "arguments": {"device": "vda", "output_format": "json"}}}'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "11",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "DEVICE         PARENT     NUM    START(SECT)    SIZE(SECT)     SIZE(BYTES)   \nvda1           vda        1      2048           124997632      63998787584"
      }
    ]
  }
}
```

</details>

