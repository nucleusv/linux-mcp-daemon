# Exec

**Tool Name**: `docker/exec`

Runs one command inside a running container and returns its stdout, stderr and exit code. No TTY, no interactive attach, no follow - one command, one answer, output capped at 1 MiB and a 30s default timeout (300s maximum; a timed-out exec keeps running inside the container, the Engine API has no way to cancel one). `command` is an argv array, not a shell line: `["sh", "-c", "ls /app"]` to use a shell. This is the sharpest tool in the group - arbitrary code, as root by default, inside whatever it targets - so its `containers:` list is separate from every other docker tool's. Which containers it may touch comes from the `containers:` list in this user's grant, not from the call. This tool always runs as root (the Docker socket is root-owned), so it takes no `privileged` argument - being able to call it at all means it was granted in mcp-sudo.yaml.

Grant this one narrowly. A container in `docker/exec`'s `containers:` list is a container this user may run arbitrary code in, as whatever user the image runs as - usually root - which is a different thing from being allowed to restart it. Keep the list shorter than `docker/manage`'s, and use `user:` in the call when the command does not need the image's default user.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

Trailing words are the argv, so a plain command needs no quoting:

```bash
linuxctl exec docker web-1 ls /usr/share/nginx/html
```

Output (Docker-in-Docker test host):
```text
Container web-1 (e36aeba4369a): ls /usr/share/nginx/html
Exit code: 0

--- stdout ---
50x.html
data
index.html
```

Shell syntax - a pipe, a redirect, `&&` - needs a shell, which means a JSON argv (a single argument the local shell will not split):

```bash
linuxctl exec docker web-1 '["sh","-c","nginx -v 2>&1; id -un"]'
```

Output:
```text
Container web-1 (e36aeba4369a): sh -c nginx -v 2>&1; id -un
Exit code: 0

--- stdout ---
nginx version: nginx/1.31.6
root
```

A non-zero exit is an answer, not an error - the exit code and stderr are reported:

```bash
linuxctl exec docker web-1 cat /etc/nope
```

Output:
```text
Container web-1 (e36aeba4369a): cat /etc/nope
Exit code: 1

--- stderr ---
cat: can't open '/etc/nope': No such file or directory
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "docker/exec", "arguments": {"container": "web-1", "command": ["sh", "-c", "nginx -v 2>&1; id -un"]}}}'

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
        "text": "Container web-1 (e36aeba4369a): sh -c nginx -v 2>&1; id -un\nExit code: 0\n\n--- stdout ---\nnginx version: nginx/1.31.6\nroot\n"
      }
    ]
  }
}
```

</details>
