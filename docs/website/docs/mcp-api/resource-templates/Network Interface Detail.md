# Network Interface Detail

**URI Template**: `network://interfaces/{name}`

Detailed properties and RX/TX traffic statistics of a specific network interface.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource network://interfaces/eth0
```

Output (Ubuntu 24.04 VPS):
```json
{
  "addresses": [
    "203.0.113.117/24",
    "fe80::216:3cff:fe43:d371/64"
  ],
  "flags": "up|broadcast|multicast|running",
  "index": 2,
  "mac": "00:16:3c:43:d3:71",
  "mtu": 1500,
  "name": "eth0",
  "statistics": {
    "rx_bytes": 3587472995,
    "rx_dropped": 0,
    "rx_errors": 0,
    "rx_packets": 3680882,
    "tx_bytes": 3449167247,
    "tx_dropped": 0,
    "tx_errors": 0,
    "tx_packets": 3703963
  }
}
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "network://interfaces/eth0"}}'
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
        "text": "{\n  \"addresses\": [\n    \"203.0.113.117/24\",\n    \"fe80::216:3cff:fe43:d371/64\"\n  ],\n  \"flags\": \"up|broadcast|multicast|running\",\n  \"index\": 2,\n  \"mac\": \"00:16:3c:43:d3:71\",\n  \"mtu\": 1500,\n  \"name\": \"eth0\",\n  \"statistics\": {\n    \"rx_bytes\": 3587482318,\n    \"rx_dropped\": 0,\n    \"rx_errors\": 0,\n    \"rx_packets\": 3680926,\n    \"tx_bytes\": 3449175439,\n    \"tx_dropped\": 0,\n    \"tx_errors\": 0,\n    \"tx_packets\": 3703996\n  }\n}",
        "uri": "network://interfaces/eth0"
      }
    ]
  }
}
```

</details>

