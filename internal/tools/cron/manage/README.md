# manage

Reads, lists and replaces crontabs: the commands cron runs for an account on a schedule. **Writing a crontab schedules commands as that account, and the job outlives the session and the revocation of the token.**

## Overview

`cron/manage` is one tool with three behaviours, chosen by its arguments:

- **Read** (no `content`): your own crontab as raw text, exactly as `crontab -l` prints it (empty when you have none).
- **List** (`privileged: true`, no `user`, no `content`): a table `USER LINES MODIFIED` of the accounts that have a crontab and that your grant lets you view.
- **Write** (`content` present): the WHOLE crontab is replaced; an empty string clears it. The `crontab` command rejects text it cannot parse and the old crontab then stays; a missing final newline is added; the limit is 64 KiB (`MaxContent`).

The crontab itself is read and written with the `crontab` command, a documented exception to "no CLI wrapping": a user cannot write their own spool file, and the command checks the syntax, applies `/etc/cron.allow` / `/etc/cron.deny` and tells cron. For systemd timers use `timers/list`.

## Who may do what

The master decides, before any worker starts (`internal/rpc/cron.go`, `prepareCronCall`); the worker only carries out one already-authorized action described by reserved `_` arguments (`_mode`, `_target_user`, `_view_users`, `_info`) that a caller's own values can never reach.

| Call | Needs | Worker runs as |
|---|---|---|
| your own crontab (no `user`, or `user` = you), read or write | nothing | you |
| list | `privileged: true` and `allowed: true` in your `cron/manage` grant | root (reads the spool; the table shows only accounts you may `view`) |
| another account's crontab, read | `privileged: true` and a rule `users: {name: {view: true}}` | root, which starts `crontab` as that account |
| another account's crontab, write | `privileged: true` and a rule `users: {name: {edit: true}}` | same |

Rules name accounts one by one (no wildcards). `edit` implies `view`. `root`'s crontab can be viewed but never edited: an `edit` rule for root is a config load error, and the worker refuses uid 0 under any name. Because the worker runs as the target account, `cron.allow` / `cron.deny` still apply to it. There is no `crontab -u <name>`: the argument list is fixed (`-l`, or `-` with the content on stdin), never a shell.

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

## Output

- Default: the raw crontab text (no newline added), or the list table, or for a write `Crontab of <user> replaced: <n> lines, sha256 <new> (was <old>)`.
- `output_format: json|yaml|table|wide` (all return the same JSON) on a read: `{user, exists, lines, jobs, bytes, sha256, content}`. `lines` counts like `crontab -l` (the spool header is stripped); `jobs` counts non-comment, non-assignment lines.
- `if_match`: pass the `sha256` you read; the write is refused with `the crontab changed since you read it (its sha256 is now ..., you passed ...) - read it again` if someone changed it meanwhile.

## Resource template

`crontab://{user}/text` (raw text) and `crontab://{user}/info` (the JSON above, without `content`) go through the same preparation and the same rules, so there is no separate `resources:` grant. See `docs/website/docs/mcp-api/resource-templates/Crontab.md`.

## Audit and errors

Every write is an audit-log line: caller, target, size, line count and the content's sha256, never the content (crontabs hold secrets). Errors name the missing piece, e.g. `another account's crontab needs privileged: true and a rule for <name>`, `user X may not edit Y's crontab: the cron/manage grant gives Y view only`, `crontab not found on this host - the cron package is not installed here`. Inside a container a call sees the container's own crontabs.
