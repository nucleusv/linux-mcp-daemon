# Crontab

**URI Template**: `crontab://{user}/{view}`

A user's crontab: `crontab://<user>/text` is the raw text exactly as `crontab -l` prints it, `crontab://<user>/info` is the metadata as JSON (`user`, `exists`, `lines`, `jobs`, `bytes`, `sha256`, without the content). With no view it defaults to `text`. It is read-only; to change a crontab use the [`cron/manage`](../tools/cron/manage) tool.

There is **no separate `resources:` grant**. A read goes through the same preparation as the tool, so one rule set decides who sees whose crontab: your own needs nothing; another account's needs `privileged` access and a `users: {name: {view: true}}` rule in your `cron/manage` grant ([mcp-sudo.yaml](../../configuration/mcp-sudo)). Another account's crontab is read by a root worker acting for that rule. Crontabs can hold secrets (passwords typed into commands); `view` is a real permission.

## Example

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource crontab://testuser/info
```

Output (Ubuntu 24.04 VPS):
```json
{
  "user": "testuser",
  "exists": true,
  "lines": 4,
  "jobs": 2,
  "bytes": 145,
  "sha256": "8448591d0f0be91fef72f7b0068a96c0c0c7c743c47472b02b2ee94ce6398858"
}
```

```bash
linuxctl resource crontab://testuser/text
```
```text
# FR-026 live test - removed by the script
MAILTO=ops@example.com
30 2 * * * /usr/local/bin/backup.sh
*/15 * * * * /usr/local/bin/healthcheck.sh
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```json
{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "crontab://testuser/info"}}
```

</details>
