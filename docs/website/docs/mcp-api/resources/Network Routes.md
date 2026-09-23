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

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "network://routes"}}'
```

Real response (captured live):
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
