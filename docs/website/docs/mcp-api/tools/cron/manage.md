# Manage

**Tool Name**: `cron/manage`

Reads or replaces a user's crontab, the list of commands cron runs for that account on a schedule, and lists which accounts have one. Without `content` it reads: your own crontab as raw text exactly as `crontab -l` prints it (empty when you have none), or with `privileged: true` and no `user`, a table of the accounts that have a crontab and that your grant lets you view. With `content` it writes: the WHOLE crontab is replaced (an empty string clears it); the `crontab` command rejects a file it cannot parse and then the old crontab is unchanged; a missing final newline is added; the limit is 64 KiB. For systemd timers use [`timers/list`](../timers/list). Needs the `crontab` command on the host; inside a container a call sees the container's own crontabs.

:::danger Writing a crontab schedules commands as that user
The job outlives the session and the revocation of the token. Every write is audit-logged (size, lines, hash, never the content). Read the [crontab risks](../../../configuration/permissions-and-risks#crontabs) before you grant `edit` on any account.
:::

## Arguments

| Argument | Type | Description |
| --- | --- | --- |
| `user` | string | Account whose crontab to read or replace; default is your own. Another account needs `privileged: true` and a rule for it |
| `content` | string | The complete new crontab; present = write (empty string clears it), absent = read. Standard crontab syntax: `m h dom mon dow command`, `@daily`, and `NAME=value` lines |
| `if_match` | string | sha256 of the crontab as you read it; the write is refused if it has changed since |
| `output_format` | string | `json` (also `yaml`/`table`/`wide`, which return the same JSON) returns `{user, exists, lines, jobs, bytes, sha256, content}` for a read; default is the raw text |
| `privileged` | boolean | Needed to list crontabs or to act on another account (needs a `cron/manage` grant); ignored for your own crontab |

## Who may do what

| Call | Needs | Worker runs as |
| --- | --- | --- |
| your own crontab, read or write | nothing | you |
| list (`privileged: true`, no `user`) | `allowed: true`; the table shows only accounts you may `view` | root (reads the spool) |
| another account, read | `privileged: true` and `users: {name: {view: true}}` | root, which starts `crontab` as that account |
| another account, write | `privileged: true` and `users: {name: {edit: true}}` | same |

Rules name accounts one by one; there are no wildcards. `edit` implies `view`. `root`'s crontab can be viewed but never edited (an `edit` rule for root is refused when the config loads). `/etc/cron.allow` and `/etc/cron.deny` still apply to the target account. Grant syntax: [mcp-sudo.yaml](../../../configuration/mcp-sudo).

```yaml
users:
  ops:
    tools:
      cron/manage:
        allowed: true
        users:
          root:      {view: true}
          test_user: {view: true, edit: true}
```

## Example

The same data is available as the [`crontab://{user}/{view}`](../../resource-templates/Crontab) resource template, under the same rules.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get crontabs                                   # my own crontab, as `crontab -l` prints it
linuxctl get crontabs --output json                     # {user, exists, lines, jobs, bytes, sha256, content}
linuxctl update crontabs --content "$(cat mycron.txt)"  # replace my whole crontab
linuxctl edit crontabs                                  # open in $EDITOR; refused if it changed meanwhile

linuxctl get crontabs --privileged true                 # accounts I may view that have a crontab
linuxctl get crontabs test_user --privileged true
linuxctl update crontabs test_user --privileged true --content "$(cat test_user.cron)"
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

Open the SSE stream and POST to the endpoint it announces (see the [MCP API overview](../../overview)), with:

```json
{"jsonrpc": "2.0", "id": "1", "method": "tools/call",
 "params": {"name": "cron/manage", "arguments": {"user": "test_user", "privileged": true,
            "content": "*/5 * * * * /usr/local/bin/backup.sh\n", "if_match": "<sha256 you read>"}}}
```

</details>
