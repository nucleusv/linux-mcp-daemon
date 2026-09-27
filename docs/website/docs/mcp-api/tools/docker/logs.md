# Logs

**Tool Name**: `docker/logs`

Reads one container's logs - the shape logs/journal-control has for systemd units. Both streams interleaved in the order Docker recorded them (`stdout: false` or `stderr: false` drops one), last 100 lines unless `lines` says otherwise, and never a follow: this returns what is there and ends. Answers are capped at 1 MiB; ask for a smaller tail rather than a bigger answer. Which containers it may touch comes from the `containers:` list in this user's grant, not from the call. This tool always runs as root (the Docker socket is root-owned), so it takes no `privileged` argument - being able to call it at all means it was granted in mcp-sudo.yaml.

Two things about the Engine API worth knowing before reading an answer:

- **`lines` is applied before the stream filter.** Docker tails first, then drops the stream that was turned off, so `lines: 4` with `stdout: false` returns however many of those last four lines came from stderr - not four stderr lines. Ask for a larger tail when filtering one stream.
- **`since` and `until` take a unix timestamp**, the form Docker's own API accepts. They are declared as strings so a timestamp survives JSON without precision games; `linuxctl --since 1759005000` is coerced for you.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get docker logs logger-1 --lines 6 --timestamps
```

Output (Docker-in-Docker test host - a container writing one line to each stream every two seconds):
```text
2026-09-27T10:10:55.842295012Z tick 10452
2026-09-27T10:10:55.842295220Z warn 10452
2026-09-27T10:10:57.843251513Z tick 10453
2026-09-27T10:10:57.843251429Z warn 10453
2026-09-27T10:10:59.845572930Z tick 10454
2026-09-27T10:10:59.845638139Z warn 10454
```

The same call with stdout dropped shows the tail-then-filter order described above - six lines were tailed, two of them were stderr:

```bash
linuxctl get docker logs logger-1 --lines 4 --stdout=false
```

Output:
```text
warn 10453
warn 10454
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "docker/logs", "arguments": {"container": "logger-1", "lines": 6, "timestamps": true}}}'

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
        "text": "2026-09-27T10:10:55.842295012Z tick 10452\n2026-09-27T10:10:55.842295220Z warn 10452\n2026-09-27T10:10:57.843251513Z tick 10453\n2026-09-27T10:10:57.843251429Z warn 10453\n2026-09-27T10:10:59.845572930Z tick 10454\n2026-09-27T10:10:59.845638139Z warn 10454\n"
      }
    ]
  }
}
```

</details>
