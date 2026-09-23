# Connections

**Tool Name**: `network/connections`

Lists active network connections and listening ports. Hint: For physical network links and IPs, use the network://interfaces resource.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get network connections
```

Output:
```text
Netid State  Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
udp   UNCONN 0      0         127.0.0.11:45825      0.0.0.0:*          
tcp   LISTEN 0      4096       127.0.0.1:44439      0.0.0.0:*          
tcp   LISTEN 0      4096      172.19.0.7:2380       0.0.0.0:*          
tcp   LISTEN 0      4096      172.19.0.7:2379       0.0.0.0:*          
tcp   LISTEN 0      4096       127.0.0.1:2379       0.0.0.0:*          
tcp   LISTEN 0      4096       127.0.0.1:2381       0.0.0.0:*          
tcp   LISTEN 0      4096       127.0.0.1:10248      0.0.0.0:*          
tcp   LISTEN 0      4096       127.0.0.1:10249      0.0.0.0:*          
tcp   LISTEN 0      4096       127.0.0.1:10259      0.0.0.0:*          
tcp   LISTEN 0      4096       127.0.0.1:10257      0.0.0.0:*          
tcp   LISTEN 0      4096      127.0.0.11:37707      0.0.0.0:*          
tcp   LISTEN 0      4096
...
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "network/connections", "arguments": {}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "17",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Netid State  Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess\nudp   UNCONN 0      0         127.0.0.11:45825      0.0.0.0:*          \ntcp   LISTEN 0      4096       127.0.0.1:44439      0.0.0.0:*          \ntcp   LISTEN 0      4096      172.19.0.7:2380       0.0.0.0:*          \ntcp   LISTEN 0      4096      172.19.0.7:2379       0.0.0.0:*          \ntcp   LISTEN 0      4096       127.0.0.1:2379       0.0.0.0:*          \ntcp   LISTEN 0      4096       127.0.0.1:2381       0.0.0.0:*          \ntcp   LISTEN 0      4096       127.0.0.1:10248      0.0.0.0:*          \ntcp   LISTEN 0      4096       127.0.0.1:10249      0.0.0.0:*          \ntcp   LISTEN 0      4096       127.0.0.1:10259      0.0.0.0:*          \ntcp   LISTEN 0      4096       127.0.0.1:10257      0.0.0.0:*          \ntcp   LISTEN 0      4096      127.0.0.11:37707      0.0.0.0:*          \ntcp   LISTEN 0      4096\n..."
      }
    ]
  }
}
```

</details>

