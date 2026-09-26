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

Output (Ubuntu 24.04 VPS):
```text
traceroute to 1.1.1.1 (1.1.1.1), 5 hops max, 60 byte packets
 1  2.57.243.2 (2.57.243.2)  0.772 ms  0.620 ms  0.588 ms
 2  212.237.216.242 (212.237.216.242)  5.186 ms  5.178 ms  5.203 ms
 3  162.158.236.14 (162.158.236.14)  18.799 ms  18.785 ms  18.806 ms
 4  162.158.236.11 (162.158.236.11)  1.470 ms  1.486 ms  1.482 ms
 5  one.one.one.one (1.1.1.1)  1.012 ms  1.072 ms  1.017 ms
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
  "id": "1",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "traceroute to 1.1.1.1 (1.1.1.1), 5 hops max, 60 byte packets\n 1  2.57.243.2 (2.57.243.2)  1.526 ms  1.472 ms  1.481 ms\n 2  212.237.216.242 (212.237.216.242)  0.664 ms  0.634 ms  0.644 ms\n 3  162.158.236.14 (162.158.236.14)  1.410 ms  1.423 ms  1.439 ms\n 4  162.158.236.11 (162.158.236.11)  1.883 ms  1.902 ms 162.158.236.13 (162.158.236.13)  12.654 ms\n 5  one.one.one.one (1.1.1.1)  1.130 ms  1.106 ms  1.085 ms\n"
      }
    ]
  }
}
```

</details>

