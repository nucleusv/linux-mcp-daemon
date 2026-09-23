# Top

**Tool Name**: `processes/top`

A snapshot like `top -b -n 1`, read natively from `/proc` - no `top` or `ps` binary. The header shows the time, uptime, logged-in users, load average, task counts by state, CPU breakdown and memory/swap in MiB; below it, the process table with all of top's columns: `PID USER PR NI VIRT RES SHR S %CPU %MEM TIME+ COMMAND`.

`%CPU` is measured the way top measures it: CPU time used over a short sampling interval (default 1 second), not an average since the process started. Like top, it is per core, so a busy multi-threaded process can exceed 100%.

| Parameter | Description |
|---|---|
| `sort_by` | `cpu` (default, like top), `mem`/`res` (resident memory), `time` (total CPU time), `pid` |
| `limit` | Maximum processes to list (default: all) |
| `user` | Only this user's processes |
| `interval_ms` | `%CPU` sampling interval in ms (default 1000, max 10000) |
| `output_format` | `json`/`yaml`/`table`/`wide` return `{"summary": ..., "processes": [...]}`, including each process's full `cmdline`, `ppid` and `threads` |
| `privileged` | Run as root |

Use [`processes/list`](./list) for a plain listing, and [`processes/delete`](./delete) to signal a process.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get processes top --limit 8
```

Output (Ubuntu 24.04 VPS, 2 vCPUs):
```text
top - 00:05:53 up  5:23,  1 user,  load average: 0.02, 0.19, 0.43
Tasks: 141 total,   2 running, 139 sleeping,   0 stopped,   0 zombie
%Cpu(s):  0.9 us,  1.8 sy,  0.0 ni, 87.7 id,  0.0 wa,  0.0 hi,  0.0 si,  9.6 st
MiB Mem :   1892.6 total,    475.7 free,    393.3 used,   1023.6 buff/cache
MiB Swap:   3772.0 total,   3739.3 free,     32.7 used.   1304.0 avail Mem

    PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
  89528 privile+  20   0 1268100  11864   6552 R  10.0   0.6   0:00.14 mcpd
  54706 root      20   0  331300   9264   7700 R   1.0   0.5   1:25.59 NetworkManager
      1 root      20   0   22596  12560   9148 S   0.0   0.6   2:13.24 systemd
      2 root      20   0       0      0      0 S   0.0   0.0   0:00.05 kthreadd
      3 root      20   0       0      0      0 S   0.0   0.0   0:00.00 pool_workqueue_release
      4 root       0 -20       0      0      0 I   0.0   0.0   0:00.00 kworker/R-rcu_g
      5 root       0 -20       0      0      0 I   0.0   0.0   0:00.00 kworker/R-rcu_p
      6 root       0 -20       0      0      0 I   0.0   0.0   0:00.00 kworker/R-slub_
```

</details>

<details>
<summary><b>linuxctl (JSON)</b></summary>

```bash
linuxctl get processes top --limit 1 -o json
```

Output:
```json
{
  "summary": {
    "time": "00:05:57",
    "uptime": "up  5:23",
    "uptime_seconds": 19432,
    "users": 1,
    "load_average": [
      0.02,
      0.18,
      0.43
    ],
    "tasks": {
      "total": 141,
      "running": 1,
      "sleeping": 140,
      "stopped": 0,
      "zombie": 0
    },
    "cpu_percent": {
      "us": 0.4,
      "sy": 1.3,
      "ni": 0,
      "id": 89,
      "wa": 0,
      "hi": 0,
      "si": 0,
      "st": 9.3
    },
    "mem_mib": {
      "total": 1892.6,
      "free": 475.7,
      "used": 393.3,
      "buff_cache": 1023.6
    },
    "swap_mib": {
      "total": 3772,
      "free": 3739.3,
      "used": 32.7,
      "avail_mem": 1304
    }
  },
  "processes": [
    {
      "pid": 89533,
      "user": "privileged",
      "pr": "20",
      "ni": 0,
      "virt_kib": 1268092,
      "res_kib": 11704,
      "shr_kib": 6552,
      "s": "R",
      "cpu_percent": 6,
      "mem_percent": 0.6,
      "time_plus": "0:00.09",
      "command": "mcpd",
      "cmdline": "/usr/local/bin/mcpd worker processes/top",
      "ppid": 89520,
      "threads": 4
    }
  ]
}
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
# 1. Open the SSE stream (in the background) and capture the one-time POST endpoint
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

# 2. POST the tools/call request to that endpoint
curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "processes/top", "arguments": {"limit": 8}}}'

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
        "text": "top - 00:05:53 up  5:23,  1 user,  load average: 0.02, 0.19, 0.43\nTasks: 141 total,   2 running, 139 sleeping,   0 stopped,   0 zombie\n%Cpu(s):  0.9 us,  1.8 sy,  0.0 ni, 87.7 id,  0.0 wa,  0.0 hi,  0.0 si,  9.6 st\nMiB Mem :   1892.6 total,    475.7 free,    393.3 used,   1023.6 buff/cache\nMiB Swap:   3772.0 total,   3739.3 free,     32.7 used.   1304.0 avail Mem\n\n    PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND\n  89528 privile+  20   0 1268100  11864   6552 R  10.0   0.6   0:00.14 mcpd\n  54706 root      20   0  331300   9264   7700 R   1.0   0.5   1:25.59 NetworkManager\n      1 root      20   0   22596  12560   9148 S   0.0   0.6   2:13.24 systemd\n      2 root      20   0       0      0      0 S   0.0   0.0   0:00.05 kthreadd\n      3 root      20   0       0      0      0 S   0.0   0.0   0:00.00 pool_workqueue_release\n      4 root       0 -20       0      0      0 I   0.0   0.0   0:00.00 kworker/R-rcu_g\n      5 root       0 -20       0      0      0 I   0.0   0.0   0:00.00 kworker/R-rcu_p\n      6 root       0 -20       0      0      0 I   0.0   0.0   0:00.00 kworker/R-slub_\n"
      }
    ]
  }
}
```

</details>
