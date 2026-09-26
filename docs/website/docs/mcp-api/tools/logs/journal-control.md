# Journal-Control

**Tool Name**: `logs/journal-control`

Queries the systemd journal (`journalctl` equivalent).

## Parameters

- `unit` (optional) - filter by systemd unit (e.g. `cron.service`).
- `lines` (optional) - number of most recent lines to return. Defaults to 100.
- `since` / `until` (optional) - time range filters (e.g. `"1 hour ago"`, `"yesterday"`, `"12:00"`).
- `reverse` (optional) - newest entries first.
- `boot` (optional) - restrict to the current boot (`journalctl -b`).
- `boot_offset` (optional) - select a prior boot relative to the current one, e.g. `-1` for the previous boot; implies `boot`.
- `output_format` (optional) - `json` for structured journal entries; omit for plain text.
- `privileged` (optional, **required in containerized deployments**) - run as root and join the host mount namespace. `journalctl` and the journal it reads are architecturally host-only and are never bundled into this daemon's own container image; without `privileged: true` this call fails with `journalctl: executable file not found in $PATH`.

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

