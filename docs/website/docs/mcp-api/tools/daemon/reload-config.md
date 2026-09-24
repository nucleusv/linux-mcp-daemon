# Reload-Config

**Tool Name**: `daemon/reload-config`

Re-reads mcpd's config files - `daemon.yaml`, `users.yaml` and `mcp-sudo.yaml` - and applies them without restarting the daemon: users and tokens, per-user grants, rate limits and tool timeouts. It takes no arguments and returns what changed.

- **It only reads the files.** They are edited on the host - with `linuxctl create|update|delete mcpd user` or `linuxctl edit mcpd config`, which call this tool afterwards themselves. There is deliberately no MCP tool that writes them.
- **Validated first, strictly.** If any file is invalid - including a key mcpd doesn't know, such as `path:` for `paths:` - nothing changes, the daemon keeps running on the config it has, and the error is returned.
- **Takes effect at once.** A new user can connect right away; a removed user, or one whose token changed, can't - and their open sessions are closed. Cached tool and resource results are dropped, since they may have been produced under grants that no longer apply.
- **Needs a restart:** `server.port`, `server.tls` and `worker.containerized`. When they changed, the reply lists them under "Need a restart to take effect".
- **Only for users granted it** in `mcp-sudo.yaml` (`daemon/reload-config: {allowed: true}`); for everyone else it isn't in `tools/list`. `scripts/install.sh` grants it to the first user. See [mcp-sudo.yaml](../../../configuration/mcp-sudo#applying-config-changes-daemonreload-config).

The daemon logs every reload with the calling user, e.g. `[CONFIG] reloaded by user=privileged: 2 change(s): user testuser: added; grants testuser: + disks/usage (root; paths [/var])`. Token values and hashes never appear in the reply or the log.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl reload daemon
```

Output, after adding `testuser` and granting it `disks/usage`:
```text
Reloaded configs/daemon.yaml, configs/users.yaml and configs/mcp-sudo.yaml.
Changes:
  user testuser: added
  grants testuser: + disks/usage (root; paths [/var])
```

With nothing changed since the last load:
```text
Reloaded configs/daemon.yaml, configs/users.yaml and configs/mcp-sudo.yaml.
No changes.
```

With an invalid file - the running config stays as it was:
```text
config not reloaded, the current one stays in effect: configs/users.yaml: yaml: line 1: did not find expected node content
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "daemon/reload-config", "arguments": {}}}'

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
        "text": "Reloaded configs/daemon.yaml, configs/users.yaml and configs/mcp-sudo.yaml.\nChanges:\n  user testuser: added\n  grants testuser: + disks/usage (root; paths [/var])\n"
      }
    ]
  }
}
```

</details>
