# Process Introspection

**URI Template**: `process://{pid}/{target}`

Reads process metadata from procfs. Valid targets: `status` (state, memory, uid/gid, threads, capabilities), `cmdline` (argv as a JSON array), `environ` (environment as a JSON object), `limits` (rlimits - max open files, max processes, etc. - as a JSON array of `{name, soft, hard, units}`), `open_files` (the process's fd table - files, sockets, pipes - as a JSON array of `{fd, target}`). Hint: Find PIDs using the processes/list tool first.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource process://1/status
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "process://1/status"}}'
```

</details>
