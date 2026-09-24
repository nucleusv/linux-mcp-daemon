# Sudo-Rules

**Tool Name**: `auth/sudo-rules`

Returns your authorized tools and privileges from mcp-sudo.yaml.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get auth sudo-rules
```

Output:
```text
Your authorized privileged tools:
{
  "Tools": {
    "auth/sudo-rules": {
      "Allowed": true,
      "Paths": null
    },
    "cpu/list": {
      "Allowed": true,
      "Paths": null
    },
    "cpu/load-average": {
      "Allowed": true,
      "Paths": null
    },
    "disks/free": {
      "Allowed": true,
      "Paths": null
    },
    "disks/health": {
      "Allowed": true,
      "Paths": null
    },
    "disks/list": {
      "Allowed": true,
      "Paths": null
    },
    "disks/mounts": {
      "Allowed": true,
      "Paths": null
    },
    "disks/partitions": {
      "Allowed": true,
      "Paths": null
    },
    "disks/performance": {
      "Allowed": true,
      "Paths": null
    },
    "disks/usage": {
      "Allowed": true,
      "Paths": null
    },
    "files/content": {
      "Allowed": true,
      "Paths": null
    },
    "files/create": {
      "Allowed": true,
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "auth/sudo-rules", "arguments": {}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "30",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Your authorized privileged tools:\n{\n  \"Tools\": {\n    \"auth/sudo-rules\": {\n      \"Allowed\": true,\n      \"Paths\": null\n    },\n    \"cpu/list\": {\n      \"Allowed\": true,\n      \"Paths\": null\n    },\n    \"cpu/load-average\": {\n      \"Allowed\": true,\n      \"Paths\": null\n    },\n    \"disks/free\": {\n      \"Allowed\": true,\n      \"Paths\": null\n    },\n    \"disks/health\": {\n      \"Allowed\": true,\n      \"Paths\": null\n    },\n    \"disks/list\": {\n      \"Allowed\": true,\n      \"Paths\": null\n    },\n    \"disks/mounts\": {\n      \"Allowed\": true,\n      \"Paths\": null\n    },\n    \"disks/partitions\": {\n      \"Allowed\": true,\n      \"Paths\": null\n    },\n    \"disks/performance\": {\n      \"Allowed\": true,\n      \"Paths\": null\n    },\n    \"disks/usage\": {\n      \"Allowed\": true,\n      \"Paths\": null\n    },\n    \"files/content\": {\n      \"Allowed\": true,\n      \"Paths\": null\n    },\n    \"files/create\": {\n      \"Allowed\": true,\n..."
      }
    ]
  }
}
```

</details>

