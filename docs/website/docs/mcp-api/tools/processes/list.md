# List

**Tool Name**: `processes/list`

Lists processes (PID, PPID, user, state, RSS, command) read from `/proc`. Read-only. Use it to find a PID, filter by `user` or one `pid`, or sort by memory. For the CPU/memory header and %CPU on every row use `processes/top`; for the process that owns a port `network/connections`; to signal a process `processes/delete`; for deep per-PID metrics the `process://<pid>/<target>` resource. Sorted by PID unless `sort_by` is `mem` (RSS, largest first) or `cpu` (samples for 0.5 s, so the call takes at least that long, and only then does `cpu_percent` appear); `limit` applies after sorting. Kernel threads appear as `[name]`. Command lines can contain secrets passed as arguments. Text output is a table; `output_format: json` returns an array of objects (pid, user, comm, state, ppid, rss_bytes, cmdline, cpu_percent). Sizes are bytes unless `human_readable`.

Text output is a table like `ps` (`PID PPID USER STAT RSS COMMAND`); RSS is in **bytes** by default, `human_readable: true` prints it in powers of 1024.

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `human_readable` | boolean | no | RSS like 10Mi; default is bytes |
| `limit` | integer | no | Maximum rows returned, applied after sorting |
| `output_format` | string | no | Use json for structured output (yaml, table and wide return the same JSON); default is text |
| `pid` | integer | no | Filter to a single specific PID |
| `privileged` | boolean | no | Run as root. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused |
| `sort_by` | string | no | pid (default), mem (RSS, largest first) or cpu (samples for 0.5 s) |
| `user` | string | no | Exact username |

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
