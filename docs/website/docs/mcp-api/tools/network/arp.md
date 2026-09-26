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

Output (Ubuntu 24.04 VPS, first 25 lines):
```text
[
  {
    "ip_address": "172.29.172.2",
    "hw_type": "0x1",
    "flags": "0x2",
    "hw_address": "52:db:73:89:f9:26",
    "mask": "*",
    "device": "amn0"
  },
  {
    "ip_address": "172.17.0.3",
    "hw_type": "0x1",
    "flags": "0x2",
    "hw_address": "6a:14:de:e7:cd:85",
    "mask": "*",
    "device": "docker0"
  },
  {
    "ip_address": "172.29.172.4",
    "hw_type": "0x1",
    "flags": "0x0",
    "hw_address": "00:00:00:00:00:00",
    "mask": "*",
    "device": "amn0"
  },
...
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
# 1. Open the SSE stream (in the background) and capture the one-time POST endpoint
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

# 2. POST the tools/call request to that endpoint
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "network/arp", "arguments": {}}}'

# 3. The result arrives on the SSE stream opened in step 1
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
        "text": "[\n  {\n    \"ip_address\": \"172.29.172.2\",\n    \"hw_type\": \"0x1\",\n    \"flags\": \"0x2\",\n    \"hw_address\": \"52:db:73:89:f9:26\",\n    \"mask\": \"*\",\n    \"device\": \"amn0\"\n  },\n  {\n    \"ip_address\": \"172.17.0.3\",\n    \"hw_type\": \"0x1\",\n    \"flags\": \"0x2\",\n    \"hw_address\": \"6a:14:de:e7:cd:85\",\n    \"mask\": \"*\",\n    \"device\": \"docker0\"\n  },\n  {\n    \"ip_address\": \"172.29.172.4\",\n    \"hw_type\": \"0x1\",\n    \"flags\": \"0x0\",\n    \"hw_address\": \"00:00:00:00:00:00\",\n    \"mask\": \"*\",\n    \"device\": \"amn0\"\n  },\n..."
      }
    ]
  }
}
```

</details>

