# List

**Tool Name**: `users/list`

Lists local user accounts from /etc/passwd and /etc/group (uid, gid, home, shell, supplementary group memberships), sorted by uid. Read-only. Never reads /etc/shadow - this reports account identity, not credentials. Local files only: LDAP/SSSD users are not listed. `min_uid` (e.g. 1000) hides system accounts. Text lines look like `name (uid=N gid=N(group)) home=... shell=... groups=a,b`; `output_format: json` returns an array of objects (username, uid, gid, group_name, comment, home_dir, shell, groups). For who logged in use `logs/logins`, for your own root grants `auth/sudo-rules`.

When `mcpd` runs containerized, passing `privileged: true` automatically lists the real host's users instead of the daemon's own container's - see [Master Daemon Configuration](../../../configuration/daemon.md)'s `worker.containerized` setting.

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `min_uid` | integer | no | Only include users with UID >= this value (e.g. 1000 to exclude system accounts) |
| `output_format` | string | no | Use json for structured output (yaml, table and wide return the same JSON); default is text |
| `privileged` | boolean | no | Run as root. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused |

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
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

# 2. POST the tools/call request to that endpoint
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
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

