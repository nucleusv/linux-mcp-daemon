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

Output:
```text
[
  {
    "iface": "eth0",
    "destination": "00000000",
    "gateway": "010013AC",
    "flags": "0003",
    "ref_cnt": "0",
    "use": "0",
    "metric": "0",
    "mask": "00000000",
    "mtu": "0",
    "window": "0",
    "irtt": "0"
  },
  {
    "iface": "vethfdafeed9",
    "destination": "0200F40A",
    "gateway": "00000000",
    "flags": "0005",
    "ref_cnt": "0",
    "use": "0",
    "metric": "0",
    "mask": "FFFFFFFF",
    "mtu": "0",
    "window": "0",
    "irtt": "0"
  },
  {
    "iface": "veth785ec824",
    "destination": "0300F40A",
    "gateway": "00000000",
    "flags": "0005",
    "ref_cnt": "0",
    "use": "0",
    "metric": "0",
    "mask": "FFFFFFFF",
    "mtu": "0",
    "window": "0",
    "irtt": "0"
  },
  {
    "iface": "veth5eefa300",
    "destination": "0400F40A",
    "gateway": "00000000",
    "flags": "0005",
    "ref_cnt": "0",
    "use": "0",
    "metric": "0"
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
  "id": "37",
  "result": {
    "contents": [
      {
        "mimeType": "application/json",
        "text": "[\n  {\n    \"iface\": \"eth0\",\n    \"destination\": \"00000000\",\n    \"gateway\": \"010013AC\",\n    \"flags\": \"0003\",\n    \"ref_cnt\": \"0\",\n    \"use\": \"0\",\n    \"metric\": \"0\",\n    \"mask\": \"00000000\",\n    \"mtu\": \"0\",\n    \"window\": \"0\",\n    \"irtt\": \"0\"\n  },\n  {\n    \"iface\": \"vethfdafeed9\",\n    \"destination\": \"0200F40A\",\n    \"gateway\": \"00000000\",\n    \"flags\": \"0005\",\n    \"ref_cnt\": \"0\",\n    \"use\": \"0\",\n    \"metric\": \"0\",\n    \"mask\": \"FFFFFFFF\",\n    \"mtu\": \"0\",\n    \"window\": \"0\",\n    \"irtt\": \"0\"\n  },\n  {\n    \"iface\": \"veth785ec824\",\n    \"destination\": \"0300F40A\",\n    \"gateway\": \"00000000\",\n    \"flags\": \"0005\",\n    \"ref_cnt\": \"0\",\n    \"use\": \"0\",\n    \"metric\": \"0\",\n    \"mask\": \"FFFFFFFF\",\n    \"mtu\": \"0\",\n    \"window\": \"0\",\n    \"irtt\": \"0\"\n  },\n  {\n    \"iface\": \"veth5eefa300\",\n    \"destination\": \"0400F40A\",\n    \"gateway\": \"00000000\",\n    \"flags\": \"0005\",\n    \"ref_cnt\": \"0\",\n    \"use\": \"0\",\n    \"metric\": \"0\"\n...",
        "uri": "network://routes"
      }
    ]
  }
}
```

</details>

