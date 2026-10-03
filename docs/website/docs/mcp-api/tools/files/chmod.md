# Chmod

**Tool Name**: `files/chmod`

Changes a file's or directory's permission bits (chmod). Mutating and idempotent; for owner or group use `files/chown`, to check the result `files/list`. Never follows symbolic links: a path containing a symlink in any component is refused, and recursive changes skip symlinks and report them. Numeric modes follow GNU chmod semantics (on directories a 4-digit mode keeps setuid/setgid; use 5 digits, e.g. 00755, to set them exactly); a bare `755` is octal. Changing a file you do not own needs `privileged: true` (a grant, and a `paths:` entry for root). Single change returns `PATH: 0644 (-rw-r--r--) -> 0755 (-rwxr-xr-x)`, or `... unchanged` if already set. A recursive run prints one line per changed entry, then `changed N, unchanged M` (plus skipped symlinks) and per-entry errors; it is not atomic, so partial success is possible.

**Symlinks are never followed.** The path is resolved one component at a time with `openat(O_PATH|O_NOFOLLOW)`, each step relative to the directory the previous step opened, and the mode is changed on exactly the object that was checked. A symlink anywhere in the path - the target itself or any directory on the way - is refused. With `recursive`, symlinks inside the tree are skipped and listed, never followed. This keeps a `privileged: true` chmod from being redirected by a planted symlink (say `/tmp/x -> /etc/shadow`) to a file outside the user's allowed paths.

Modes follow GNU chmod exactly (verified in tests across many modes), including its directory rule: a numeric mode of up to four digits keeps a directory's setuid/setgid bits - use five digits (`00755`) to set them exactly. `X` adds execute only to directories and already-executable files.

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `mode` | string | yes | Octal (644, 0755, 4755) or symbolic (u+x, go-w, a=r, +X, u+s, +t; comma-separated) |
| `path` | string | yes | Absolute path |
| `privileged` | boolean | no | Run as root - needed for files you don't own. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path |
| `recursive` | boolean | no | Also apply to everything below a directory (symlinks are skipped, never followed) |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl chmod files /tmp/demo/deploy.sh u+x
```

Output:
```text
/tmp/demo/deploy.sh: 0644 (-rw-r--r--) -> 0744 (-rwxr--r--)
```

</details>

<details>
<summary><b>linuxctl (recursive - the symlink inside is skipped)</b></summary>

```bash
linuxctl chmod files /tmp/demo go-rwx --recursive true
```

Output:
```text
/tmp/demo: 0755 (drwxr-xr-x) -> 0700 (drwx------)
/tmp/demo/deploy.sh: 0744 (-rwxr--r--) -> 0700 (-rwx------)
/tmp/demo/logs: 0755 (drwxr-xr-x) -> 0700 (drwx------)
/tmp/demo/logs/app.log: 0644 (-rw-r--r--) -> 0600 (-rw-------)
changed 4, unchanged 0, skipped 1 symlink(s) (never followed): /tmp/demo/passwd-link
```

</details>

<details>
<summary><b>linuxctl (a symlink target is refused)</b></summary>

```bash
linuxctl chmod files /tmp/demo/passwd-link 0600
```

Output:
```text
/tmp/demo/passwd-link: refusing to follow a symbolic link
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "files/chmod", "arguments": {"path": "/tmp/demo/deploy.sh", "mode": "u+x"}}}'

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
        "text": "/tmp/demo/deploy.sh: 0644 (-rw-r--r--) -> 0744 (-rwxr--r--)\n"
      }
    ]
  }
}
```

</details>
