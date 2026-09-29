# System-Control

**Tool Name**: `kernel/system-control`

Reads or writes a kernel parameter (sysctl) at runtime through /proc/sys. With `value` omitted it reads: `key` (dotted `net.ipv4.ip_forward` or slash form; a directory such as `net.ipv4` prints its subtree) returns `key = value` lines, and `read_all: true` (only when `key` is empty) prints every parameter, thousands of lines, uncapped. Reading needs no grant. With `value` it WRITES: that needs `privileged: true` and a grant, is refused for keys outside the user's `sysctl.write_keys` globs in mcp-sudo.yaml, and the reply shows the value the kernel now holds. Writes last until reboot; nothing is persisted to /etc/sysctl.d. For OS and kernel version use `system/os-release`, for memory figures `memory/usage`.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get kernel sysctl net.ipv4.ip_forward
```

Output:
```text
net.ipv4.ip_forward = 1
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "kernel/system-control", "arguments": {"key": "net.ipv4.ip_forward"}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "24",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "net.ipv4.ip_forward = 1"
      }
    ]
  }
}
```

</details>

