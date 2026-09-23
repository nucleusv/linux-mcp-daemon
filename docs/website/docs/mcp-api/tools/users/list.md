# List

**Tool Name**: `users/list`

Lists user accounts from `/etc/passwd` (uid, gid, home, shell, group memberships). Never reads `/etc/shadow` - this reports account identity, not credentials.

When `mcpd` runs containerized, passing `privileged: true` automatically lists the real host's users instead of the daemon's own container's - see [Master Daemon Configuration](../../../configuration/daemon.md)'s `worker.containerized` setting.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get users --min_uid 1000
```

Output:
```text
ubuntu (uid=1000 gid=1000) home=/home/ubuntu shell=/bin/bash
testuser (uid=1001 gid=1001) home=/home/testuser shell=/bin/bash
unpriviliged (uid=1002 gid=1002) home=/home/unpriviliged shell=/bin/bash
privileged (uid=1003 gid=1003) home=/home/privileged shell=/bin/bash
nobody (uid=65534 gid=65534) home=/nonexistent shell=/usr/sbin/nologin
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "users/list", "arguments": {"min_uid": 1000}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "29",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "ubuntu (uid=1000 gid=1000) home=/home/ubuntu shell=/bin/bash\ntestuser (uid=1001 gid=1001) home=/home/testuser shell=/bin/bash\nunpriviliged (uid=1002 gid=1002) home=/home/unpriviliged shell=/bin/bash\nprivileged (uid=1003 gid=1003) home=/home/privileged shell=/bin/bash\nnobody (uid=65534 gid=65534) home=/nonexistent shell=/usr/sbin/nologin"
      }
    ]
  }
}
```

</details>

