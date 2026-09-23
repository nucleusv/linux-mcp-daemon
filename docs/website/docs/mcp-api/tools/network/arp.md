# Arp

**Tool Name**: `arp`

Displays the ARP cache.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get network arp
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
# 1. Open the SSE stream (in the background) and capture the one-time POST endpoint
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

# 2. POST the tools/call request to that endpoint
curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "network/arp", "arguments": {}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

</details>

**Real response (captured live):**

```text
[
  {
    "ip_address": "172.19.0.8",
    "hw_type": "0x1",
    "flags": "0x2",
    "hw_address": "d6:ee:9a:61:0c:f8",
    "mask": "*",
    "device": "eth0"
  },
  {
    "ip_address": "10.244.0.3",
    "hw_type": "0x1",
    "flags": "0x2",
    "hw_address": "d2:f4:ff:a6:64:23",
    "mask": "*",
    "device": "veth785ec824"
  },
  {
    "ip_address": "172.19.0.3",
    "hw_type": "0x1",
    "flags": "0x2",
    "hw_address": "02:82:72:c2:cd:d8",
    "mask": "*",
    "device": "eth0"
  },
  {
    "ip_address": "10.244.0.2",
    "hw_type": "0x1",
    "flags": "0x2",
    "hw_address": "c6:a2:80:2b:82:36",
    "mask": "*",
    "device": "vethfdafeed9"
  },
  {
    "ip_address": "10.244.0.7",
    "hw_type": "0x1",
    "flags": "0x2",
    "hw_address": "c2:73:6d:dc:17:a1",
    "mask": "*",
    "device": "vethbdcd73f8"
  },
  {
    "ip_address": "10.244.0.5",
    "hw_type": "0x1",
    "flags": "0x
...
```
