# Usage

**Tool Name**: `disks/usage`

Measures how much disk space a directory tree uses (native walk, like `du`). Read-only. For the free space of a whole filesystem use `disks/free`; to find individual big files use `files/find` with `size`. By default the reply is one grand total; `max_depth: N` also lists directories up to N levels deep, largest first; `all: true` adds a per-file list (text output only). Sizes are allocated blocks unless `apparent_size: true`; hard links count once, symlinks are never followed. `exclude` patterns containing `/` match the full path (`/proc`, `/var/lib/*`), others the base name. Unreadable directories are skipped silently, so an unprivileged total can under-count: use `privileged: true` (a grant with a `paths:` entry). Results are cached for 60 s per user and arguments. The shipped config allows 300 s, otherwise the default is 30 s. Text ends with `Total size of PATH: N`; `output_format: json` returns an object (path, total_size, human_size with `human_readable`, directory_sizes as a path-to-bytes map only when `max_depth` > 0).

Sizes are in **bytes** by default; `human_readable: true` prints them like `du -h`.

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `path` | string | yes | Absolute path of the directory to measure |
| `all` | boolean | no | Also list every file, largest first (text output only) |
| `apparent_size` | boolean | no | Report logical file sizes instead of allocated disk blocks |
| `exclude` | array of string | no | Patterns to skip: one containing '/' matches the full path (e.g. '/proc', '/var/lib/*'), others the base name (e.g. '*.tmp') |
| `human_readable` | boolean | no | Sizes like du -h (4.0 KiB); default is bytes |
| `max_depth` | integer | no | 0 or omitted: grand total only; N: also list directories up to N levels deep, largest first |
| `one_file_system` | boolean | no | Skip directories on different file systems (-x) |
| `output_format` | string | no | Use json for structured output (yaml, table and wide return the same JSON); default is text |
| `privileged` | boolean | no | Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused |
| `separate_dirs` | boolean | no | For directories do not include size of subdirectories (-S) |
| `threshold` | integer | no | Bytes: a positive value hides entries smaller than this, a negative value hides larger ones (printed lines only) |

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
