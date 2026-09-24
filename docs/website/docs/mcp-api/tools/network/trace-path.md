# Trace-Path

**Tool Name**: `network/trace-path`

Traces the network path to a host (equivalent to traceroute). Useful for debugging routing issues, identifying where packets are dropped, or measuring network latency across hops. Hint: Use network/ping for basic reachability before tracing the path.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get network trace-path 1.1.1.1 --max_hops 5
```

Output:
```text
traceroute to 1.1.1.1 (1.1.1.1), 3 hops max, 60 byte packets
 1  172.19.0.1 (172.19.0.1)  1.643 ms  0.054 ms  0.009 ms
 2  * * *
 3  * * *
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "network/trace-path", "arguments": {"host": "1.1.1.1", "max_hops": 5}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "18",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "traceroute to 1.1.1.1 (1.1.1.1), 3 hops max, 60 byte packets\n 1  172.19.0.1 (172.19.0.1)  1.643 ms  0.054 ms  0.009 ms\n 2  * * *\n 3  * * *"
      }
    ]
  }
}
```

</details>

