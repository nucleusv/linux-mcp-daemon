# Update

**Tool Name**: `files/update`

Edits part of an EXISTING file in place: appends text or replaces an inclusive line range. Mutating and not atomic; the file keeps its mode and owner. To write a whole file use `files/create`; to see the lines first use `files/read` with `start_line`/`end_line`. With `append: true` exactly `content` is added at the end (no newline is added, include your own) and the file is created if missing, though not its parent directories; append wins over any line range. Otherwise `start_line` AND `end_line` are both required (1-indexed, `end_line` >= `start_line`), the file must exist, and those lines are replaced by `content` (one trailing newline of `content` is ignored; an `end_line` past the end is clamped; a `start_line` past the end adds `content` as a new last line). There is no insert or delete mode: replacing with empty `content` leaves one empty line. Returns `Successfully appended to PATH` or `Successfully updated lines A-B in PATH`, no diff. `path` must be absolute; `privileged: true` (a grant, and for root a `paths:` entry) edits files you cannot write.

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `content` | string | yes | Text to append, or to replace the line range with (no newline is added when appending) |
| `path` | string | yes | Absolute path of the file to edit |
| `append` | boolean | no | If true, append content at the end (creates the file if missing); takes precedence over start_line/end_line |
| `end_line` | integer | no | Last line to replace (inclusive, >= start_line; past the end of the file is clamped) |
| `privileged` | boolean | no | Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused |
| `start_line` | integer | no | First line to replace (1-indexed); required together with end_line unless append is true |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl update files /tmp/note.txt --content " world" --append true
```

Output:
```text
Successfully appended to /tmp/note.txt
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "files/update", "arguments": {"path": "/tmp/note.txt", "content": " world", "append": true}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "c2",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Successfully appended to /tmp/note.txt"
      }
    ]
  }
}
```

</details>

