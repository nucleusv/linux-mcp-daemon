# Network Interfaces

**URI**: `network://interfaces`

Network interfaces, assigned IP addresses, and detailed RX/TX traffic statistics for all interfaces.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource network://interfaces
```

Output:
```text
[
  {
    "addresses": [
      "127.0.0.1/8",
      "::1/128"
    ],
    "flags": "up|loopback|running",
    "index": 1,
    "mac": "",
    "mtu": 65536,
    "name": "lo",
    "statistics": {
      "rx_bytes": 5655107440,
      "rx_dropped": 0,
      "rx_errors": 0,
      "rx_packets": 22774140,
      "tx_bytes": 5655107440,
      "tx_dropped": 0,
      "tx_errors": 0,
      "tx_packets": 22774140
    }
  },
  {
    "addresses": null,
    "flags": "0",
    "index": 2,
    "mac": "",
    "mtu": 1480,
    "name": "tunl0",
    "statistics": {
      "rx_bytes": 0,
      "rx_dropped": 0,
      "rx_errors": 0,
      "rx_packets": 0,
      "tx_bytes": 0,
      "tx_dropped": 0,
      "tx_errors": 0,
      "tx_packets": 0
    }
  },
  {
    "addresses": null,
    "flags": "0",
    "index": 3,
    "mac": "",
    "mtu": 1476,
    "name": "gre0",
    "statistics": {
      "rx_bytes": 0,
      "rx_dr
...
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "network://interfaces"}}'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "36",
  "result": {
    "contents": [
      {
        "mimeType": "application/json",
        "text": "[\n  {\n    \"addresses\": [\n      \"127.0.0.1/8\",\n      \"::1/128\"\n    ],\n    \"flags\": \"up|loopback|running\",\n    \"index\": 1,\n    \"mac\": \"\",\n    \"mtu\": 65536,\n    \"name\": \"lo\",\n    \"statistics\": {\n      \"rx_bytes\": 5655107440,\n      \"rx_dropped\": 0,\n      \"rx_errors\": 0,\n      \"rx_packets\": 22774140,\n      \"tx_bytes\": 5655107440,\n      \"tx_dropped\": 0,\n      \"tx_errors\": 0,\n      \"tx_packets\": 22774140\n    }\n  },\n  {\n    \"addresses\": null,\n    \"flags\": \"0\",\n    \"index\": 2,\n    \"mac\": \"\",\n    \"mtu\": 1480,\n    \"name\": \"tunl0\",\n    \"statistics\": {\n      \"rx_bytes\": 0,\n      \"rx_dropped\": 0,\n      \"rx_errors\": 0,\n      \"rx_packets\": 0,\n      \"tx_bytes\": 0,\n      \"tx_dropped\": 0,\n      \"tx_errors\": 0,\n      \"tx_packets\": 0\n    }\n  },\n  {\n    \"addresses\": null,\n    \"flags\": \"0\",\n    \"index\": 3,\n    \"mac\": \"\",\n    \"mtu\": 1476,\n    \"name\": \"gre0\",\n    \"statistics\": {\n      \"rx_bytes\": 0,\n      \"rx_dr\n...",
        "uri": "network://interfaces"
      }
    ]
  }
}
```

</details>

