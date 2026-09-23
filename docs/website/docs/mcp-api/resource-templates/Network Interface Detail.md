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

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "network://interfaces/eth0"}}'
```

</details>

**Real response (captured live):**

```json
{
  "addresses": [
    "172.19.0.7/16",
    "fc00:f853:ccd:e793::7/64",
    "fe80::b444:e3ff:fec0:4fd2/64"
  ],
  "flags": "up|broadcast|multicast|running",
  "index": 11,
  "mac": "b6:44:e3:c0:4f:d2",
  "mtu": 65535,
  "name": "eth0",
  "statistics": {
    "rx_bytes": 683918914,
    "rx_dropped": 0,
    "rx_errors": 0,
    "rx_packets": 890576,
    "tx_bytes": 864826874,
    "tx_dropped": 0,
    "tx_errors": 0,
    "tx_packets": 713740
  }
}
```
