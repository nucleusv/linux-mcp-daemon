# Chown

**Tool Name**: `files/chown`

Changes owner and/or group, like `chown` - natively, no `chown` binary. Changing a file's owner requires `privileged: true`.

**Symlinks are never followed** - same guarantee as [`files/chmod`](./chmod): the path is walked component by component with `openat(O_PATH|O_NOFOLLOW)`, any symlink in it is refused, and ownership changes on exactly the verified object (`fchownat(fd, "", AT_EMPTY_PATH)`). With `recursive`, symlinks inside the tree are skipped and reported.

| Parameter | Meaning |
|---|---|
| `path` | Absolute path (required) |
| `owner` | `user`, `user:group`, `:group` (group only), or `user:` (the user's login group); names or numeric ids (required) |
| `recursive` | Also everything below a directory |
| `privileged` | Run as root - required to change ownership; authorized per path in `mcp-sudo.yaml` |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl (recursive)</b></summary>

```bash
linuxctl chown files /tmp/chown-demo/site nobody:nogroup --recursive true --privileged true
```

Output:
```text
/tmp/chown-demo/site: root:root -> nobody:nogroup
/tmp/chown-demo/site/app.css: root:root -> nobody:nogroup
/tmp/chown-demo/site/index.html: root:root -> nobody:nogroup
changed 3, unchanged 0
```

</details>

<details>
<summary><b>linuxctl (group only)</b></summary>

```bash
linuxctl chown files /tmp/chown-demo/site/index.html :root --privileged true
```

Output:
```text
/tmp/chown-demo/site/index.html: nobody:nogroup -> nobody:root


$ linuxctl get files list /tmp/chown-demo/site --privileged true
total 8
-rw-r--r-- 1 nobody nogroup 1 Sep 23 23:42 app.css
-rw-r--r-- 1 nobody root    5 Sep 23 23:44 index.html
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
# 1. Open the SSE stream (in the background) and capture the one-time POST endpoint
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

# 2. POST the tools/call request to that endpoint
curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "files/chown", "arguments": {"path": "/tmp/chown-demo/site", "owner": "nobody:nogroup", "recursive": true, "privileged": true}}}'

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
        "text": "/tmp/chown-demo/site: root:root -> nobody:nogroup\n/tmp/chown-demo/site/app.css: root:root -> nobody:nogroup\n/tmp/chown-demo/site/index.html: root:root -> nobody:nogroup\nchanged 3, unchanged 0\n"
      }
    ]
  }
}
```

</details>
