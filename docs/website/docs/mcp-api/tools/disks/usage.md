# Usage

**Tool Name**: `disks/usage`

Calculates the disk space used by a directory, like du -s - in bytes, or like du -sh with human_readable. Use disks/free for overall partition stats.

| Parameter | Description |
|---|---|
| `path` | Directory to measure (required) |
| `max_depth` | List subdirectories down to this depth, largest first (`0`: the total only) |
| `human_readable` | Sizes like `du -h` (`75.8 MiB`); default is bytes |
| `apparent_size` | Apparent sizes instead of disk usage (`--apparent-size`) |
| `all` | Count files too, not only directories (`-a`) |
| `separate_dirs` | Don't include subdirectories in a directory's size (`-S`) |
| `one_file_system` | Stay on one filesystem (`-x`) |
| `exclude` | Patterns to skip |
| `threshold` | Skip entries smaller than this many bytes (negative: larger than) |
| `output_format` | `json`/`yaml`/`table`/`wide`: sizes in bytes |
| `privileged` | Run as root to read protected subdirectories (authorized per path in `mcp-sudo.yaml`) |

Sizes are in **bytes** by default; `human_readable: true` prints them like `du -h`.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get disks usage /var/log --max_depth 1 --privileged true
```

Output (Ubuntu 24.04 VPS, 2 GB RAM):
```text
Directory sizes (up to depth 1, largest first):
79605760	/var/log
64258048	/var/log/journal
966656	/var/log/installer
376832	/var/log/apt
16384	/var/log/unattended-upgrades
4096	/var/log/chrony
4096	/var/log/dist-upgrade
4096	/var/log/private
4096	/var/log/sysstat
---
Total size of /var/log: 79605760
```

</details>

<details>
<summary><b>linuxctl (human_readable)</b></summary>

```bash
linuxctl get disks usage /var/log --max_depth 1 --privileged true --human_readable true
```

Output:
```text
Directory sizes (up to depth 1, largest first):
75.9 MiB	/var/log
61.3 MiB	/var/log/journal
944.0 KiB	/var/log/installer
368.0 KiB	/var/log/apt
16.0 KiB	/var/log/unattended-upgrades
4.0 KiB	/var/log/chrony
4.0 KiB	/var/log/dist-upgrade
4.0 KiB	/var/log/private
4.0 KiB	/var/log/sysstat
---
Total size of /var/log: 75.9 MiB
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "disks/usage", "arguments": {"path": "/var/log", "max_depth": 1, "privileged": true}}}'

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
        "text": "Directory sizes (up to depth 1, largest first):\n79605760\t/var/log\n64258048\t/var/log/journal\n966656\t/var/log/installer\n376832\t/var/log/apt\n16384\t/var/log/unattended-upgrades\n4096\t/var/log/chrony\n4096\t/var/log/dist-upgrade\n4096\t/var/log/private\n4096\t/var/log/sysstat\n---\nTotal size of /var/log: 79605760\n"
      }
    ]
  }
}
```

</details>
