# Find

**Tool Name**: `files/find`

Searches a directory tree (default `/`) by name, type, age or size, like `find`. Read-only; never follows symlinks; skips `/proc`, `/sys`, `/dev` and `/run` when the search starts at `/` or above them (not when `path` is inside one). Use `files/list` to see one known directory and `disks/usage` to see which folders take the space. All filters are ANDed and none is required. There is NO result cap: `path: /` without a filter lists every file on the host, so give `name`, `type` or `max_depth`; the 30 s worker timeout applies. Syntax: `name` is a case-sensitive glob on the base name only (`*.log`, not a path); `type` is one of `f d l b c p s` or a comma list (`f,d`); `mtime` in days: `+7` older than 7 days, `-1` within the last day, `7` exactly 7 days; `size`: `+100M` larger, `-10k` smaller (units b c w k M G, rounded up); `max_depth` 1 = direct children, omitted or 0 = unlimited. Unreadable directories are skipped silently. Text output: one `SIZE PATH` line per match, sorted by path (no size for directories; empty output = no match). `output_format: json` returns an array of objects (path, size in bytes, type).

Results are sorted by path, with each file's size in **bytes** by default; `human_readable: true` prints sizes in powers of 1024.

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `human_readable` | boolean | no | Text output only: sizes like 1.5 KiB; default is bytes |
| `max_depth` | integer | no | Levels below path to descend (1 = direct children); omit or 0 for unlimited |
| `mtime` | string | no | Days since modification: '+7' older than 7 days, '-1' within the last day, '7' exactly 7 days |
| `name` | string | no | Glob on the file's base name only, case-sensitive, e.g. '*.log' (not a path pattern) |
| `output_format` | string | no | json (yaml, table and wide return the same JSON) gives an array of objects with path, size, type; default is text |
| `path` | string | no | Absolute starting directory (default /). A large tree with no filter can take longer than the 30 s worker limit |
| `privileged` | boolean | no | Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused |
| `size` | string | no | '+100M' larger than 100 MiB, '-10k' smaller than 10 KiB; units b c w k M G, rounded up to the unit |
| `type` | string | no | f file, d directory, l symlink, b, c, p, s; or a comma list such as 'f,d' |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get files find /var/log --name "*.log" --privileged true
```

Output (Ubuntu 24.04 VPS, 2 GB RAM):
```text
33531      /var/log/alternatives.log
0          /var/log/apport.log
94761      /var/log/apt/history.log
243110     /var/log/apt/term.log
4537003    /var/log/auth.log
42214      /var/log/boot.log
61229      /var/log/bootstrap.log
4739       /var/log/cloud-init-output.log
84179      /var/log/cloud-init.log
1031647    /var/log/dpkg.log
6696       /var/log/installer/block/discover.log
3346       /var/log/installer/cloud-init-output.log
56132      /var/log/installer/cloud-init.log
113137     /var/log/installer/curtin-install.log
31         /var/log/installer/subiquity-client-debug.log
30         /var/log/installer/subiquity-client-info.log
31         /var/log/installer/subiquity-server-debug.log
30         /var/log/installer/subiquity-server-info.log
335030     /var/log/kern.log
5427       /var/log/unattended-upgrades/unattended-upgrades-dpkg.log
0          /var/log/unattended-upgrades/unattended-upgrades-shutdown.log
1133       /var/log/unattended-upgrades/unattended-upgrades.log
```

</details>

<details>
<summary><b>linuxctl (human_readable)</b></summary>

```bash
linuxctl get files find /var/log --name "*.log" --privileged true --human_readable true
```

Output:
```text
32.7 KiB   /var/log/alternatives.log
0 B        /var/log/apport.log
92.5 KiB   /var/log/apt/history.log
237.4 KiB  /var/log/apt/term.log
4.3 MiB    /var/log/auth.log
41.2 KiB   /var/log/boot.log
59.8 KiB   /var/log/bootstrap.log
4.6 KiB    /var/log/cloud-init-output.log
82.2 KiB   /var/log/cloud-init.log
1007.5 KiB /var/log/dpkg.log
6.5 KiB    /var/log/installer/block/discover.log
3.3 KiB    /var/log/installer/cloud-init-output.log
54.8 KiB   /var/log/installer/cloud-init.log
110.5 KiB  /var/log/installer/curtin-install.log
31 B       /var/log/installer/subiquity-client-debug.log
30 B       /var/log/installer/subiquity-client-info.log
31 B       /var/log/installer/subiquity-server-debug.log
30 B       /var/log/installer/subiquity-server-info.log
327.2 KiB  /var/log/kern.log
5.3 KiB    /var/log/unattended-upgrades/unattended-upgrades-dpkg.log
0 B        /var/log/unattended-upgrades/unattended-upgrades-shutdown.log
1.1 KiB    /var/log/unattended-upgrades/unattended-upgrades.log
```

</details>

<details>
<summary><b>linuxctl (JSON)</b></summary>

```bash
linuxctl get files find /var/log --name auth.log --privileged true -o json
```

Output:
```json
[
  {
    "path": "/var/log/auth.log",
    "size": 4537003,
    "type": "f"
  }
]
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "files/find", "arguments": {"path": "/var/log", "name": "*.log", "privileged": true}}}'

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
        "text": "33531      /var/log/alternatives.log\n0          /var/log/apport.log\n94761      /var/log/apt/history.log\n243110     /var/log/apt/term.log\n4537003    /var/log/auth.log\n42214      /var/log/boot.log\n61229      /var/log/bootstrap.log\n4739       /var/log/cloud-init-output.log\n84179      /var/log/cloud-init.log\n1031647    /var/log/dpkg.log\n6696       /var/log/installer/block/discover.log\n3346       /var/log/installer/cloud-init-output.log\n56132      /var/log/installer/cloud-init.log\n113137     /var/log/installer/curtin-install.log\n31         /var/log/installer/subiquity-client-debug.log\n30         /var/log/installer/subiquity-client-info.log\n31         /var/log/installer/subiquity-server-debug.log\n30         /var/log/installer/subiquity-server-info.log\n335030     /var/log/kern.log\n5427       /var/log/unattended-upgrades/unattended-upgrades-dpkg.log\n0          /var/log/unattended-upgrades/unattended-upgrades-shutdown.log\n1133       /var/log/unattended-upgrades/unattended-upgrades.log\n"
      }
    ]
  }
}
```

</details>
