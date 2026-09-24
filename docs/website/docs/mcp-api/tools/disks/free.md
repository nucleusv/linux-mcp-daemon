# Free

**Tool Name**: `disks/free`

Returns disk space statistics (df -h). Use disks/list to see all block devices.

| Parameter | Description |
|---|---|
| `path` | Absolute path; reports the filesystem it lives on (required) |
| `human_readable` | Sizes like `53.2 GiB` (`df -h`); default is bytes |
| `inodes` | Inode counts instead of block usage (`df -i`) |
| `output_format` | `json`/`yaml`/`table`/`wide`: `total_bytes`, `used_bytes`, `free_bytes`, `use_percent`, plus `*_human` fields with `human_readable` |
| `privileged` | Run as root (authorized per path in `mcp-sudo.yaml`) |

Sizes are in **bytes** by default; `human_readable: true` prints them in powers of 1024.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get disks free /
```

Output (Ubuntu 24.04 VPS, 2 GB RAM):
```text
Filesystem space on /
Total:     57078390784
Used:      12853035008 (22.5%)
Available: 44225355776
```

</details>

<details>
<summary><b>linuxctl (human_readable)</b></summary>

```bash
linuxctl get disks free / --human_readable true
```

Output:
```text
Filesystem space on /
Total:     53.2 GiB
Used:      12.0 GiB (22.5%)
Available: 41.2 GiB
```

</details>

<details>
<summary><b>linuxctl (JSON, human_readable)</b></summary>

```bash
linuxctl get disks free / --human_readable true -o json
```

Output:
```json
{
  "free_bytes": 44225351680,
  "free_human": "41.2 GiB",
  "path": "/",
  "total_bytes": 57078390784,
  "total_human": "53.2 GiB",
  "use_percent": 22.51822261885301,
  "used_bytes": 12853039104,
  "used_human": "12.0 GiB"
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "disks/free", "arguments": {"path": "/"}}}'

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
        "text": "Filesystem space on /\nTotal:     57078390784\nUsed:      12853035008 (22.5%)\nAvailable: 44225355776\n"
      }
    ]
  }
}
```

</details>
