# Create

**Tool Name**: `files/create`

Writes a whole file: creates it, or REPLACES the content of an existing one entirely. Mutating and not atomic. Missing parent directories are created (mode 0755); a new file gets mode 0644, an existing file keeps its mode. With `content` omitted or empty it only touches: creates an empty file or refreshes the modification time and never truncates an existing file. To change part of an existing file use `files/update` (append or replace lines); to check first whether a path exists use `files/list` or `files/find`; afterwards set mode or owner with `files/chmod`/`files/chown`. `content` is text and no trailing newline is added; `path` must be absolute. Writing where your account cannot needs `privileged: true` (a grant, and for root a `paths:` entry covering the path, which also refuses symlinks in the path; otherwise a symlink at the path is followed). Returns one line, `Successfully created and wrote to PATH` or `Successfully touched PATH`; failures are plain text such as `failed to write to file`.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl create files /tmp/note.txt --content hello
```

Output:
```text
Successfully created and wrote to /tmp/note.txt
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "files/create", "arguments": {"path": "/tmp/note.txt", "content": "hello"}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "c1",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Successfully created and wrote to /tmp/note.txt"
      }
    ]
  }
}
```

</details>

