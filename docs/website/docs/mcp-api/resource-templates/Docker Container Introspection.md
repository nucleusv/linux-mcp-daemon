# Docker Container Introspection

**URI Template**: `docker-container://{name}/{view}`

One Docker container's own view of itself, chosen with `{view}`:

| View | Shows |
|---|---|
| `status` | Computed summary - state, health, exit code, restart count, uptime, image, ports, limits |
| `inspect` | Docker's full raw config - everything `docker inspect` prints |
| `stats` | One CPU/memory/network/IO snapshot (not a stream) |
| `top` | The processes running inside the container |

Hint: list containers with the docker/containers tool first.

A read of this template needs two grants in [mcp-sudo.yaml](../../configuration/mcp-sudo.md): the template itself (`docker-container://{name}/{view}`) and the internal worker it spawns (`docker/inspect`). The `containers:` list on the template's grant is enforced the same way `docker/manage`'s is - against the name and the resolved ID, before the socket is dialled.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

`status` is the computed view - the one worth reading first. Docker reports a start timestamp and a wall of config; this reports the state, the uptime derived from it, and only the limits actually set:

```bash
linuxctl resource docker-container://web-1/status
```

Output (Docker-in-Docker test host):
```json
{
  "created": "2026-09-27T10:04:49.776969426Z",
  "exit_code": 0,
  "id": "e36aeba4369a725068b13142ef53c541abdfbaf857cfdc4b8370210e095a2099",
  "image": "nginx:alpine",
  "limits": {
    "cpus": 1,
    "memory_bytes": 268435456,
    "memory_swap_bytes": 536870912
  },
  "name": "web-1",
  "networks": {
    "bridge": "172.18.0.2"
  },
  "oom_killed": false,
  "paused": false,
  "pid": 12895,
  "ports": [
    "0.0.0.0:8080->80/tcp",
    ":::8080->80/tcp"
  ],
  "privileged": false,
  "restart_count": 0,
  "restart_policy": {
    "maximum_retry_count": 0,
    "name": "unless-stopped"
  },
  "restarting": false,
  "running": true,
  "started_at": "2026-09-27T10:11:09.264561962Z",
  "state": "running",
  "uptime": "9s"
}
```

A container with a healthcheck also reports `health` and `health_failing_streak`; a stopped one reports `finished_at` instead of `uptime`.

`top` is the process table from inside the container, keyed by Docker's own column titles:

```bash
linuxctl resource docker-container://web-1/top
```

Output:
```json
{
  "processes": [
    {
      "command": "nginx: master process nginx -g daemon off;",
      "pid": "12895",
      "time": "0:00",
      "user": "root"
    },
    {
      "command": "nginx: worker process",
      "pid": "12947",
      "time": "0:00",
      "user": "101"
    },
    {
      "command": "nginx: worker process",
      "pid": "12948",
      "time": "0:00",
      "user": "101"
    },
    {
      "command": "nginx: worker process",
      "pid": "12949",
      "time": "0:00",
      "user": "101"
    },
    {
      "command": "nginx: worker process",
      "pid": "12950",
      "time": "0:00",
      "user": "101"
    }
  ],
  "titles": [
    "PID",
    "USER",
    "TIME",
    "COMMAND"
  ]
}
```

`stats` is one snapshot - `stream=false`, so it returns and ends rather than holding a connection open. Docker's own metrics, 100 lines of them; abridged here:

```bash
linuxctl resource docker-container://web-1/stats
```

Output (abridged - the full answer includes every cgroup memory counter, `blkio_stats` and `precpu_stats`):
```json
{
  "id": "e36aeba4369a725068b13142ef53c541abdfbaf857cfdc4b8370210e095a2099",
  "name": "/web-1",
  "os_type": "linux",
  "read": "2026-09-27T10:11:27.399974846Z",
  "cpu_stats": {
    "cpu_usage": {
      "total_usage": 20249000,
      "usage_in_kernelmode": 9591000,
      "usage_in_usermode": 10657000
    },
    "system_cpu_usage": 198384610000000,
    "online_cpus": 4,
    "throttling_data": {
      "periods": 2,
      "throttled_periods": 0,
      "throttled_time": 0
    }
  },
  "memory_stats": {
    "usage": 4866048,
    "limit": 268435456
  },
  "networks": {
    "eth0": {
      "rx_bytes": 740,
      "rx_packets": 10,
      "rx_errors": 0,
      "rx_dropped": 0,
      "tx_bytes": 126,
      "tx_packets": 3,
      "tx_errors": 0,
      "tx_dropped": 0
    }
  },
  "pids_stats": {
    "current": 5,
    "limit": 18446744073709551615
  }
}
```

A CPU percentage needs two samples: the snapshot carries `precpu_stats` alongside `cpu_stats`, which is the previous reading Docker took one second earlier.

`inspect` is Docker's full raw config for the container - 250 lines for this nginx container, everything `docker inspect` prints. Read `status` unless a specific field is actually needed.

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "docker-container://web-1/top"}}'
```

Response (abridged to two of the five processes):
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "contents": [
      {
        "mimeType": "application/json",
        "text": "{\n  \"processes\": [\n    {\n      \"command\": \"nginx: master process nginx -g daemon off;\",\n      \"pid\": \"12895\",\n      \"time\": \"0:00\",\n      \"user\": \"root\"\n    },\n    {\n      \"command\": \"nginx: worker process\",\n      \"pid\": \"12947\",\n      \"time\": \"0:00\",\n      \"user\": \"101\"\n    }\n  ],\n  \"titles\": [\n    \"PID\",\n    \"USER\",\n    \"TIME\",\n    \"COMMAND\"\n  ]\n}",
        "uri": "docker-container://web-1/top"
      }
    ]
  }
}
```

</details>
