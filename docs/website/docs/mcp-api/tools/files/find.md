# Find

**Tool Name**: `files/find`

Search for files in a directory hierarchy.

| Parameter | Description |
|---|---|
| `path` | Directory to search (default `/`) |
| `name` | Glob for file names, e.g. `*.log` |
| `type` | `f` file, `d` directory, `l` symlink |
| `size` | Size filter, e.g. `+100M` (larger than 100 MiB) |
| `mtime` | Modification time filter, e.g. `+7` (older than 7 days) |
| `max_depth` | How deep to descend |
| `human_readable` | Sizes like `4.3 MiB`; default is bytes |
| `output_format` | `json`/`yaml`/`table`/`wide`: `path`, `type`, `size` (bytes) |
| `privileged` | Search as root (authorized per path in `mcp-sudo.yaml`) |

Results are sorted by path, with each file's size in **bytes** by default; `human_readable: true` prints sizes in powers of 1024.

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
