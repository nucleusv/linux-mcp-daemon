# Journal-Control

**Tool Name**: `logs/journal-control`

Reads the systemd journal (wraps `journalctl -n LINES --no-pager`). Read-only. Returns the last `lines` entries (default 100, the only size limit), oldest first unless `reverse`. Filter with `unit` (`sshd.service`), `since`/`until` in journalctl syntax (`1 hour ago`, `yesterday`, `2026-09-29 10:00`), `boot: true` (current boot) or `boot_offset` (-1 = previous boot; takes precedence over `boot`). Without root an ordinary user sees only their own entries unless in the `systemd-journal` or `adm` group. Requires privileged: true in containerized deployments (needs a grant), since journalctl only exists on the host, never in this daemon's own image. Plain text by default; `output_format: json` gives one JSON object per line (journalctl -o json), not an array. For kernel messages use `logs/dmesg`, for logins `logs/logins`, for container output `docker/logs`, for unit state `services/list`.

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `boot` | boolean | no | Restrict output to the current boot (journalctl -b) |
| `boot_offset` | integer | no | Select a prior boot relative to the current one, e.g. -1 for the previous boot (implies boot) |
| `lines` | integer | no | Number of entries to tail (default 100; the only size limit) |
| `output_format` | string | no | json returns one JSON object per line (journalctl -o json); default is plain text |
| `privileged` | boolean | no | Run as root and join the host mount namespace - required in containerized deployments (needs a grant for this tool in mcp-sudo.yaml) |
| `reverse` | boolean | no | Output newest entries first |
| `since` | string | no | journalctl time syntax, e.g. '1 hour ago', 'today', '2026-09-29 10:00' |
| `unit` | string | no | Filter by systemd unit (e.g., 'kubelet.service') |
| `until` | string | no | Filter logs until a specific time (e.g., 'yesterday', '12:00') |

In a containerized deployment `journalctl` and the journal it reads are host-only and are never bundled into this daemon's image; without `privileged: true` the call fails with `journalctl: executable file not found in $PATH`.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get logs journal --unit cron.service --lines 3 --privileged true
```

Output (Ubuntu 24.04 VPS):
```text
Sep 26 08:05:01 vps.example.com CRON[29847]: pam_unix(cron:session): session opened for user root(uid=0) by root(uid=0)
Sep 26 08:05:01 vps.example.com CRON[29848]: (root) CMD (command -v debian-sa1 > /dev/null && debian-sa1 1 1)
Sep 26 08:05:01 vps.example.com CRON[29847]: pam_unix(cron:session): session closed for user root
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "logs/journal-control", "arguments": {"unit": "cron.service", "lines": 3, "privileged": true}}}'
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
        "text": "Sep 26 08:05:01 vps.example.com CRON[29847]: pam_unix(cron:session): session opened for user root(uid=0) by root(uid=0)\nSep 26 08:05:01 vps.example.com CRON[29848]: (root) CMD (command -v debian-sa1 > /dev/null && debian-sa1 1 1)\nSep 26 08:05:01 vps.example.com CRON[29847]: pam_unix(cron:session): session closed for user root\n"
      }
    ]
  }
}
```

</details>

