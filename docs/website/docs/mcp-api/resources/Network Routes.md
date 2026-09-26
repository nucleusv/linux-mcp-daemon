# Network Routes

**URI**: `network://routes`

IPv4 Routing Table (/proc/net/route). Hint: Use network/ping to test reachability.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource network://routes
```

Output (Ubuntu 24.04 VPS, first 25 lines):
```text
[
  {
    "destination": "0.0.0.0/0",
    "gateway": "203.0.113.1",
    "iface": "eth0",
    "metric": 100,
    "flags": [
      "up",
      "gateway"
    ],
    "mtu": 0,
    "window": 0,
    "irtt": 0,
    "default": true
  },
  {
    "destination": "1.1.1.1/32",
    "gateway": "203.0.113.1",
    "iface": "eth0",
    "metric": 100,
    "flags": [
      "up",
      "gateway",
      "host"
    ],
...
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "network://routes"}}'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "contents": [
      {
        "mimeType": "application/json",
        "text": "[\n  {\n    \"destination\": \"0.0.0.0/0\",\n    \"gateway\": \"203.0.113.1\",\n    \"iface\": \"eth0\",\n    \"metric\": 100,\n    \"flags\": [\n      \"up\",\n      \"gateway\"\n    ],\n    \"mtu\": 0,\n    \"window\": 0,\n    \"irtt\": 0,\n    \"default\": true\n  },\n  {\n    \"destination\": \"1.1.1.1/32\",\n    \"gateway\": \"203.0.113.1\",\n    \"iface\": \"eth0\",\n    \"metric\": 100,\n    \"flags\": [\n      \"up\",\n      \"gateway\",\n      \"host\"\n    ],\n...",
        "uri": "network://routes"
      }
    ]
  }
}
```

</details>

