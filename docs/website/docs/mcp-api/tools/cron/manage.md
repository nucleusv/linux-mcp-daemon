# Manage

**Tool Name**: `cron/manage`

Reads or replaces a user's crontab, the list of commands cron runs for that account on a schedule, and lists which accounts have one. Without `content` it reads: your own crontab as raw text exactly as `crontab -l` prints it (empty when you have none), or with `privileged: true` and no `user`, a table of the accounts that have a crontab and that your grant lets you view. With `content` it writes: the WHOLE crontab is replaced (an empty string clears it); the `crontab` command rejects a file it cannot parse and then the old crontab is unchanged; a missing final newline is added; the limit is 64 KiB. For systemd timers use [`timers/list`](../timers/list). Needs the `crontab` command on the host; inside a container a call sees the container's own crontabs.

:::danger Writing a crontab schedules commands as that user
The job outlives the session and the revocation of the token. Every write is audit-logged (size, lines, hash, never the content). Read the [crontab risks](../../../configuration/permissions-and-risks#crontabs) before you grant `edit` on any account.
:::

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `content` | string | no | The complete new crontab; present = write (empty string clears it), absent = read. Standard crontab syntax: `m h dom mon dow command`, `@daily`, and NAME=value lines |
| `if_match` | string | no | sha256 of the crontab as you read it; the write is refused if it has changed since |
| `output_format` | string | no | 'json' (also yaml/table/wide, which return the same JSON) returns \{user, exists, lines, jobs, bytes, sha256, content\} for a read; default is the raw text |
| `privileged` | boolean | no | Needed to list crontabs or to act on another account (needs a cron/manage grant); ignored for your own crontab |
| `user` | string | no | Account whose crontab to read or replace; default is your own. Another account needs privileged: true and a rule for it |

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

Your own crontab needs no grant (output from an Ubuntu 24.04 VPS, as `fr-test`):

```bash
linuxctl update crontabs --content "$(cat mycron.txt)"
```
```text
Crontab of fr-test replaced: 4 lines, sha256 9703f4a065f38cb417e1cf8040dc24eb25332c0f919155869e2f2fc894e6bb77 (was e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855)
```
`e3b0c442...` is the hash of an empty crontab, so this account had none before.

```bash
linuxctl get crontabs
```
```text
# nightly backup
MAILTO=ops@example.com
30 2 * * * /usr/local/bin/backup.sh
*/15 * * * * /usr/local/bin/healthcheck.sh
```

```bash
linuxctl get crontabs --output json
```
```json
{
  "user": "fr-test",
  "exists": true,
  "lines": 4,
  "jobs": 2,
  "bytes": 119,
  "sha256": "9703f4a065f38cb417e1cf8040dc24eb25332c0f919155869e2f2fc894e6bb77",
  "content": "# nightly backup\nMAILTO=ops@example.com\n30 2 * * * /usr/local/bin/backup.sh\n*/15 * * * * /usr/local/bin/healthcheck.sh\n"
}
```
`jobs` counts the lines cron runs: not comments, not `NAME=value` lines.

`--output table` and `--output wide` print the same fields one per line, `--output yaml` as YAML (output from the same host, as `fr-test`):

```text
$ linuxctl get crontabs --output table
USER      fr-test
EXISTS    true
LINES     4
JOBS      2
BYTES     119
SHA256    9703f4a065f38cb417e1cf8040dc24eb25332c0f919155869e2f2fc894e6bb77
CONTENT   # nightly backup
MAILTO=ops@example.com
30 2 * * * /usr/local/bin/backup.sh
*/15 * * * * /usr/local/bin/healthcheck.sh

$ linuxctl get crontabs --output yaml
user: fr-test
exists: true
lines: 4
jobs: 2
bytes: 119
sha256: 9703f4a065f38cb417e1cf8040dc24eb25332c0f919155869e2f2fc894e6bb77
content: |
    # nightly backup
    MAILTO=ops@example.com
    30 2 * * * /usr/local/bin/backup.sh
    */15 * * * * /usr/local/bin/healthcheck.sh
```

A write with a stale `if_match` is refused and nothing changes:
```bash
linuxctl update crontabs --content "# other" --if_match 0000000000000000000000000000000000000000000000000000000000000000
```
```text
the crontab changed since you read it (its sha256 is now 9703f4a065f38cb417e1cf8040dc24eb25332c0f919155869e2f2fc894e6bb77, you passed 0) - read it again
```

`linuxctl edit crontabs` works like `crontab -e`, in your `$VISUAL`, `$EDITOR` or `vi` (arguments are allowed: `EDITOR="code --wait"`). It reads the crontab and its hash, opens a private copy (mode 0600), and writes back only if you changed it:

```text
$ EDITOR=vim linuxctl edit crontabs            # added one line, saved
Crontab of fr-test replaced: 3 lines, sha256 38e324f76ed654e162931ee1134501b30b8541e0e722031301e08e70e2a12173 (was bae7af7f9180bd04a29ad9a3ce2a2700389bb803accc3dcd804f624c91e90a4f)

$ EDITOR=vim linuxctl edit crontabs            # closed without changes
No changes to the crontab of fr-test.

$ EDITOR=vim linuxctl edit crontabs            # someone else changed it while the editor was open
the crontab changed since you read it (its sha256 is now c35594bcd30d33a8c7b5b77c26186d6e3103b2f6ee4586449f4b51bab0208b91, you passed 38e324f76ed654e162931ee1134501b30b8541e0e722031301e08e70e2a12173) - read it again
Your edit was not saved; it is kept in /tmp/crontab-1234.txt (read the crontab again, merge, and update).
```

For another account add the name and `--privileged true`: `linuxctl edit crontabs test_user --privileged true`.

Another account's crontab, as a user whose `cron/manage` grant has `testuser: {view, edit}`, `unpriviliged: {view}` and `root: {view}`:
```bash
linuxctl get crontabs --privileged true          # accounts with a crontab that you may view
```
```text
USER             LINES  MODIFIED
testuser         2      2026-10-03T13:08:22Z
```
```bash
linuxctl update crontabs testuser --privileged true --content "$(cat testuser.cron)"
```
```text
Crontab of testuser replaced: 2 lines, sha256 c81476659b4f3988d850cb8ed1d4e3cd8b0fd8f3f136604b50eb2fb5ffee3bbb (was 0a12fac05663da5bf57a142703afa5b3249857f837a3cdbd64d0dcdb4b0117e3)
```

What is refused, and what the message says:
```text
$ linuxctl update crontabs root --privileged true --content "* * * * * /bin/true"
user privileged may not edit root's crontab: the cron/manage grant gives root view only

$ linuxctl update crontabs unpriviliged --privileged true --content "* * * * * /bin/true"
user privileged may not edit unpriviliged's crontab: the cron/manage grant gives unpriviliged view only

$ linuxctl get crontabs testuser          # another account without --privileged true
another account's crontab needs privileged: true and a rule for testuser in your cron/manage grant (users: {testuser: {view: true}})

$ linuxctl get crontabs root --privileged true      # as a user with no rule for root
user testuser is not authorized to run cron/manage on another account: it needs `allowed: true` and a users: rule for root in your grant in mcp-sudo.yaml
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
