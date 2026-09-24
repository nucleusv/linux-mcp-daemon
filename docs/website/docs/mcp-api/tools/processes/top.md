# Top

**Tool Name**: `processes/top`

A snapshot like `top -b -n 1`, read natively from `/proc` - no `top` or `ps` binary. The header shows the time, uptime, logged-in users, load average, task counts by state, CPU breakdown and memory/swap - in bytes by default, in MiB like top with `human_readable`; below it, the process table with all of top's columns: `PID USER PR NI VIRT RES SHR S %CPU %MEM TIME+ COMMAND`.

`%CPU` is measured the way top measures it: CPU time used over a short sampling interval (default 1 second), not an average since the process started. Like top, it is per core, so a busy multi-threaded process can exceed 100%.

| Parameter | Description |
|---|---|
| `sort_by` | `cpu` (default, like top), `mem`/`res` (resident memory), `time` (total CPU time), `pid` |
| `limit` | Maximum processes to list (default: all) |
| `user` | Only this user's processes |
| `human_readable` | Memory like top: a `MiB Mem`/`MiB Swap` header and VIRT/RES/SHR in KiB; default is bytes |
| `interval_ms` | `%CPU` sampling interval in ms (default 1000, max 10000) |
| `output_format` | `json`/`yaml`/`table`/`wide` return `{"summary": ..., "processes": [...]}`, including each process's full `cmdline`, `ppid` and `threads`; memory in bytes (`mem_bytes`, `swap_bytes`, `virt_bytes`, `res_bytes`, `shr_bytes`) |
| `privileged` | Run as root |

Use [`processes/list`](./list) for a plain listing, and [`processes/delete`](./delete) to signal a process.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get processes top --limit 8
```

Output (Ubuntu 24.04 VPS, 2 GB RAM, 2 vCPUs):
```text
top - 21:16:06 up 22 min,  1 user,  load average: 1.62, 1.17, 0.83
Tasks: 145 total,   1 running, 144 sleeping,   0 stopped,   0 zombie
%Cpu(s):  0.9 us,  0.9 sy,  0.0 ni, 92.2 id,  0.0 wa,  0.0 hi,  0.0 si,  6.0 st
B Mem : 1984503808 total, 386248704 free, 326496256 used, 1271758848 buff/cache
B Swap: 3955224576 total, 3955224576 free, 0 used. 1452421120 avail Mem

    PID USER      PR  NI         VIRT         RES         SHR S  %CPU  %MEM     TIME+ COMMAND
   6876 privile+  20   0   1366355968    12500992     7315456 S   6.0   0.6   0:00.09 mcpd
    741 root      20   0    339189760    19329024    16289792 S   2.0   1.0   0:09.53 NetworkManager
      1 root      20   0     23105536    14053376     9785344 S   0.0   0.7   0:05.28 systemd
      2 root      20   0            0           0           0 S   0.0   0.0   0:00.01 kthreadd
      3 root      20   0            0           0           0 S   0.0   0.0   0:00.00 pool_workqueue_release
      4 root       0 -20            0           0           0 I   0.0   0.0   0:00.00 kworker/R-rcu_g
      5 root       0 -20            0           0           0 I   0.0   0.0   0:00.00 kworker/R-rcu_p
      6 root       0 -20            0           0           0 I   0.0   0.0   0:00.00 kworker/R-slub_
```

</details>

<details>
<summary><b>linuxctl (human_readable)</b></summary>

```bash
linuxctl get processes top --limit 8 --human_readable true
```

Output:
```text
top - 21:16:07 up 22 min,  1 user,  load average: 1.49, 1.15, 0.82
Tasks: 145 total,   1 running, 144 sleeping,   0 stopped,   0 zombie
%Cpu(s):  0.5 us,  1.4 sy,  0.0 ni, 89.5 id,  1.4 wa,  0.0 hi,  0.0 si,  7.3 st
MiB Mem :   1892.6 total,    367.4 free,    312.1 used,   1213.0 buff/cache
MiB Swap:   3772.0 total,   3772.0 free,      0.0 used.   1384.4 avail Mem

    PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
   6884 privile+  20   0 1268460  11688   7080 S   5.0   0.6   0:00.07 mcpd
     35 root      20   0       0      0      0 I   1.0   0.0   0:00.81 kworker/u64:2-events_power_efficient
      1 root      20   0   22564  13724   9556 S   0.0   0.7   0:05.28 systemd
      2 root      20   0       0      0      0 S   0.0   0.0   0:00.01 kthreadd
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
    "time": "21:16:08",
    "uptime": "up 22 min",
    "uptime_seconds": 1357,
    "users": 1,
    "load_average": [
      1.49,
      1.15,
      0.82
    ],
    "tasks": {
      "total": 145,
      "running": 1,
      "sleeping": 144,
      "stopped": 0,
      "zombie": 0
    },
    "cpu_percent": {
      "us": 0.9,
      "sy": 1.3,
      "ni": 0,
      "id": 88.9,
      "wa": 0,
      "hi": 0,
      "si": 0.4,
      "st": 8.4
    },
    "mem_bytes": {
      "total": 1984503808,
      "free": 384827392,
      "used": 327725056,
      "buff_cache": 1271951360
    },
    "swap_bytes": {
      "total": 3955224576,
      "free": 3955224576,
      "used": 0,
      "avail_mem": 1451192320
    }
  },
  "processes": [
    {
      "pid": 6892,
      "user": "privileged",
      "pr": "20",
      "ni": 0,
      "virt_bytes": 1299173376,
      "res_bytes": 12177408,
      "shr_bytes": 7057408,
      "s": "S",
      "cpu_percent": 4,
      "mem_percent": 0.6,
      "time_plus": "0:00.07",
      "command": "mcpd",
      "cmdline": "/usr/local/bin/mcpd worker processes/top",
      "ppid": 4509,
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
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

# 2. POST the tools/call request to that endpoint
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
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
        "text": "top - 21:16:06 up 22 min,  1 user,  load average: 1.62, 1.17, 0.83\nTasks: 145 total,   1 running, 144 sleeping,   0 stopped,   0 zombie\n%Cpu(s):  0.9 us,  0.9 sy,  0.0 ni, 92.2 id,  0.0 wa,  0.0 hi,  0.0 si,  6.0 st\nB Mem : 1984503808 total, 386248704 free, 326496256 used, 1271758848 buff/cache\nB Swap: 3955224576 total, 3955224576 free, 0 used. 1452421120 avail Mem\n\n    PID USER      PR  NI         VIRT         RES         SHR S  %CPU  %MEM     TIME+ COMMAND\n   6876 privile+  20   0   1366355968    12500992     7315456 S   6.0   0.6   0:00.09 mcpd\n    741 root      20   0    339189760    19329024    16289792 S   2.0   1.0   0:09.53 NetworkManager\n      1 root      20   0     23105536    14053376     9785344 S   0.0   0.7   0:05.28 systemd\n      2 root      20   0            0           0           0 S   0.0   0.0   0:00.01 kthreadd\n      3 root      20   0            0           0           0 S   0.0   0.0   0:00.00 pool_workqueue_release\n      4 root       0 -20            0           0           0 I   0.0   0.0   0:00.00 kworker/R-rcu_g\n      5 root       0 -20            0           0           0 I   0.0   0.0   0:00.00 kworker/R-rcu_p\n      6 root       0 -20            0           0           0 I   0.0   0.0   0:00.00 kworker/R-slub_\n"
      }
    ]
  }
}
```

</details>
