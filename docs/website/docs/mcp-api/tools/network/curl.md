# Curl

**Tool Name**: `curl`

Makes one HTTP(S) request with Go's HTTP client and returns status, headers and body. NOT read-only: any `method` (default GET) is sent as given, so POST, PUT or DELETE change the remote system. Follows redirects (up to 10); a non-2xx status is not an error, check `status_code`. The default timeout is 10 s (`timeout`, whole seconds) and the 30 s worker limit caps anything larger. The body is cut at `max_body` (default 1 MiB, max 10 MiB) and `truncated` is then true. `insecure` skips TLS verification. The daemon's proxy environment is honored unless the user has a `network:` policy in mcp-sudo.yaml; such a policy applies to every call and redirect hop, and a blocked destination fails to connect. Header and body values are redacted in the audit log. Returns JSON with `status_code`, `status`, `headers` (values comma-joined), `body`, `truncated`; `output_format` is ignored. For DNS use `network/nslookup`, for TCP reachability `network/ping`, for local files `files/read`.

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

