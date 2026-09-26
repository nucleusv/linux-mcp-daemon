# Curl

**Tool Name**: `curl`

Executes an HTTP GET request.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get network curl http://127.0.0.1:9091/ping
```

Output (Ubuntu 24.04 VPS):
```json
{
  "status_code": 400,
  "status": "400 Bad Request",
  "headers": {},
  "body": "Client sent an HTTP request to an HTTPS server.\n"
}
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "network/curl", "arguments": {"url": "http://127.0.0.1:9091/ping"}}}'

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
        "text": "{\n  \"status_code\": 400,\n  \"status\": \"400 Bad Request\",\n  \"headers\": {},\n  \"body\": \"Client sent an HTTP request to an HTTPS server.\\n\"\n}"
      }
    ]
  }
}
```

</details>

