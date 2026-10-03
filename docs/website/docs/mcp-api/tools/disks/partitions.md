# Partitions

**Tool Name**: `disks/partitions`

Lists the partitions of a disk with start sector and size in sectors and bytes, read natively from /sys/class/block (no fdisk). Read-only. `device` names the PARENT DISK (`sda`, `nvme0n1`), not a partition; without it every disk's partitions are listed. A sector size of 512 bytes is assumed. It does not report partition type, label, UUID or filesystem: use `disks/list` or `disks/mounts` for those, and `disks/list` to find device names. A named disk without partitions is an error (`no partitions found for device`). Output is a text table; when `output_format` is set to any non-empty value (the parameter is accepted although not listed in the schema) it is an indented JSON array of objects (device, parent_disk, number, start_sector, size_sectors, size_bytes).

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `device` | string | no | Parent disk name such as 'sda' or 'nvme0n1' (not a partition); omit for all disks |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get disks partitions vda --output json
```

Output (Ubuntu 24.04 VPS):
```text
[
  {
    "device": "vda1",
    "parent_disk": "vda",
    "number": 1,
    "start_sector": 2048,
    "size_sectors": 31455232,
    "size_bytes": 16105078784
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "disks/partitions", "arguments": {"device": "vda", "output_format": "json"}}}'
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
        "text": "[\n  {\n    \"device\": \"vda1\",\n    \"parent_disk\": \"vda\",\n    \"number\": 1,\n    \"start_sector\": 2048,\n    \"size_sectors\": 31455232,\n    \"size_bytes\": 16105078784\n  }\n]"
      }
    ]
  }
}
```

</details>

