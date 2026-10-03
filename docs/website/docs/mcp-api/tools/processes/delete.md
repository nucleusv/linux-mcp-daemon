# Delete

**Tool Name**: `processes/delete`

Sends ONE signal to ONE process by PID (kill(2)); the default SIGTERM asks the process to exit. Mutating and not idempotent: it returns as soon as the signal is delivered and does not check that the process exited, and signalling a PID that is gone fails with `no such process`. Allowed `signal` values: SIGTERM, SIGKILL, SIGHUP, SIGINT, SIGQUIT, SIGUSR1, SIGUSR2, SIGSTOP, SIGCONT, SIGABRT (SIG prefix optional, case-insensitive, numbers rejected), so it can also pause (SIGSTOP) and resume (SIGCONT). Refuses PID 1 and the mcpd daemon itself. Another user's process fails with `not permitted` unless `privileged: true` (needs a grant). Find the PID first with `processes/list` or `processes/top`. To stop a managed service use `services/manage` (systemd may restart a killed one), for a container `docker/manage`. Returns one text line, `Successfully sent signal SIGTERM to process N`; `output_format` has no effect.

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `pid` | integer | yes | PID to signal (positive integer; PID 1 and mcpd itself are refused) |
| `output_format` | string | no | Ignored - the reply is always text |
| `privileged` | boolean | no | Run as root to signal other users' processes. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused |
| `signal` | string | no | SIGTERM (default), SIGKILL, SIGHUP, SIGINT, SIGQUIT, SIGUSR1, SIGUSR2, SIGSTOP, SIGCONT or SIGABRT; SIG prefix optional, case-insensitive |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl delete processes 1234 --signal SIGTERM
```

Output (not executed here - hypothetical, matches the tool's source code):
```text
Successfully sent signal SIGTERM to process 1234
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "processes/delete", "arguments": {"pid": 1234, "signal": "SIGTERM"}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response (not executed here - hypothetical, matches the tool's source code):
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Successfully sent signal SIGTERM to process 1234"
      }
    ]
  }
}
```

</details>

