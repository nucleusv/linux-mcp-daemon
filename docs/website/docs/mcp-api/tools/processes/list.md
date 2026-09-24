# List

**Tool Name**: `processes/list`

Lists running processes on the system. Use this to find a PID, then use the process://\{pid\}/\{target\} resource for deep metrics or processes/delete to kill it.

| Parameter | Description |
|---|---|
| `sort_by` | `cpu`, `mem` or `pid` |
| `limit` | Return at most this many processes |
| `user` | Only this user's processes |
| `pid` | Only this PID |
| `human_readable` | RSS like `158Mi`; default is bytes |
| `output_format` | `json`/`yaml`/`table`/`wide`: `pid`, `ppid`, `user`, `comm`, `state`, `rss_bytes`, `cmdline` (and `cpu_percent` with `sort_by: cpu`) |
| `privileged` | Run as root |

Text output is a table like `ps` (`PID PPID USER STAT RSS COMMAND`); RSS is in **bytes** by default, `human_readable: true` prints it in powers of 1024.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get processes --sort_by mem --limit 5
```

Output (Ubuntu 24.04 VPS, 2 GB RAM):
```text
PID   PPID  USER  STAT  RSS        COMMAND
1005  1     root  S     172650496  /usr/bin/dockerd -H fd:// --containerd=/run/containerd/containerd.sock
917   1     root  S     62439424   /usr/bin/containerd
386   1     root  S     27918336   /sbin/multipathd -d -s
724   1     root  S     21233664   /usr/bin/python3 /usr/bin/networkd-dispatcher --run-startup-triggers
741   1     root  S     19267584   /usr/sbin/NetworkManager --no-daemon
```

</details>

<details>
<summary><b>linuxctl (human_readable)</b></summary>

```bash
linuxctl get processes --sort_by mem --limit 5 --human_readable true
```

Output:
```text
PID   PPID  USER  STAT  RSS    COMMAND
1005  1     root  S     165Mi  /usr/bin/dockerd -H fd:// --containerd=/run/containerd/containerd.sock
917   1     root  S     60Mi   /usr/bin/containerd
386   1     root  S     27Mi   /sbin/multipathd -d -s
724   1     root  S     20Mi   /usr/bin/python3 /usr/bin/networkd-dispatcher --run-startup-triggers
741   1     root  S     18Mi   /usr/sbin/NetworkManager --no-daemon
```

</details>

<details>
<summary><b>linuxctl (JSON)</b></summary>

```bash
linuxctl get processes --sort_by mem --limit 1 -o json
```

Output:
```json
[
  {
    "pid": 1005,
    "user": "root",
    "comm": "dockerd",
    "state": "S",
    "ppid": 1,
    "rss_bytes": 172650496,
    "cmdline": "/usr/bin/dockerd -H fd:// --containerd=/run/containerd/containerd.sock"
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "processes/list", "arguments": {"sort_by": "mem", "limit": 5}}}'

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
        "text": "PID   PPID  USER  STAT  RSS        COMMAND\n1005  1     root  S     172650496  /usr/bin/dockerd -H fd:// --containerd=/run/containerd/containerd.sock\n917   1     root  S     62439424   /usr/bin/containerd\n386   1     root  S     27918336   /sbin/multipathd -d -s\n724   1     root  S     21233664   /usr/bin/python3 /usr/bin/networkd-dispatcher --run-startup-triggers\n741   1     root  S     19267584   /usr/sbin/NetworkManager --no-daemon\n"
      }
    ]
  }
}
```

</details>
