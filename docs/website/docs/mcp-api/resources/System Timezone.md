# System Timezone

**URI**: `system://timezone`

Configured IANA timezone (e.g. `America/New_York`) plus current local offset and time.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource system://timezone
```

Output:
```text
Time zone: UTC
Abbreviation: UTC
UTC offset: +0.0
Current local time: Wed, 23 Sep 2026 11:43:50 UTC
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "system://timezone"}}'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "34",
  "result": {
    "contents": [
      {
        "mimeType": "text/plain",
        "text": "Time zone: UTC\nAbbreviation: UTC\nUTC offset: +0.0\nCurrent local time: Wed, 23 Sep 2026 11:43:50 UTC",
        "uri": "system://timezone"
      }
    ]
  }
}
```

</details>

