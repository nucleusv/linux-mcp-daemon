# List

**Tool Name**: `timers/list`

Lists the systemd timers of the host - systemd's scheduler, the modern counterpart of cron - each with the unit it starts, its schedule (`OnCalendar=` and monotonic settings such as `OnBootSec=`), its next and last run, whether it is `Persistent` (catches up runs missed while the host was off) and its last result. Read-only. On many hosts timers are the only scheduler in use (the Ubuntu test VPS has 18 timers and no user crontab), and `services/list` does not show them because it keeps only `.service` units. To inspect the unit a timer starts, use `linuxctl describe system <name>` or the `service://<name>/status` resource. Times are RFC 3339 UTC and `never` means systemd reports none (a timer that has not fired yet, or one with no upcoming trigger). `pattern` accepts a leading and/or trailing `*` (`apt*`, `*.timer`, `*daily*`); the name includes `.timer`. Unprivileged calls need the host's systemd bus (a bare-metal or VM host); inside a container use `privileged: true`, which joins the host. Starting, stopping or enabling a timer is not offered by this tool.

## Arguments

| Argument | Type | Description |
| --- | --- | --- |
| `pattern` | string | Wildcard on the timer name (e.g. `apt*`, `*.timer`, `*daily*`); no other wildcards |
| `active_state` | string | Only timers in this active state (e.g. `active`, `inactive`, `failed`) |
| `output_format` | string | `json` (also `yaml`/`table`/`wide`, which return the same JSON) for an array of objects; default is text |
| `privileged` | boolean | Run as root (needed inside a container to reach the host's systemd; needs a grant) |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get timers --pattern "apt*"
```

Output (Ubuntu 24.04 VPS, as an unprivileged user):
```text
[active] apt-daily-upgrade.timer
  Next: 2026-09-30T06:10:05Z | Last: 2026-09-29T06:37:20Z | Runs: apt-daily-upgrade.service
  Schedule: OnCalendar=*-*-* 06:00:00
  Desc: Daily apt upgrade and clean activities

[active] apt-daily.timer
  Next: 2026-09-29T10:12:15Z | Last: 2026-09-28T20:40:39Z | Runs: apt-daily.service
  Schedule: OnCalendar=*-*-* 06,18:00:00
  Desc: Daily apt download activities
```

JSON, one timer:
```bash
linuxctl get timers --pattern "logrotate*" --output json
```
```json
[
  {
    "name": "logrotate.timer",
    "description": "Daily rotation of log files",
    "load_state": "loaded",
    "active_state": "active",
    "sub_state": "waiting",
    "unit": "logrotate.service",
    "next_run": "2026-09-30T00:00:00Z",
    "last_run": "2026-09-29T00:00:01Z",
    "schedule": [
      "OnCalendar=*-*-* 00:00:00"
    ],
    "persistent": true,
    "result": "success"
  }
]
```

A timer that has never fired shows `never`, and monotonic schedules are written as in a unit file:
```bash
linuxctl get timers --active_state inactive
```
```text
[inactive] apport-autoreport.timer
  Next: never | Last: never | Runs: apport-autoreport.service
  Schedule: OnUnitActiveSec=3h0m0s; OnStartupSec=1h0m0s
```

No match:
```bash
linuxctl get timers --pattern "nosuch*"
```
```text
No timers found matching the criteria.
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "timers/list", "arguments": {"pattern": "logrotate*", "output_format": "json"}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

The result's `content[0].text` is the JSON array shown above (as an escaped string).

</details>
