# FR-026 cron/manage: view and edit crontabs, with per-user view/edit rules

- **Created:** 2026-09-29, by the owner (design agreed in chat: the caller's own crontab by default, per-user rules for view and edit, no `commands:` allowlist)
- **Related:** FR-023 (research: the cron deep-dive, section (c)), FR-025 (`timers/list`), `internal/tools/docker/manage` and its `containers:` grant (the pattern), `internal/config/sudo.go`, ARCHITECTURE.md (documented CLI exceptions), FR-024 (description checklist)

## Description

One tool, **`cron/manage`** (no separate `cron/list`), `action: list | add | remove`, optional `user` (default: the caller). Same shape as `docker/manage` / `services/manage`.

- **Whose crontab:** by default the caller's own OS account. Unprivileged, no root: through the `crontab` command (`crontab -l`, `crontab -`), because a user's crontab is not writable by that user - checked on the VPS: `/var/spool/cron/crontabs` is `root:crontab` mode 1730 and only `/usr/bin/crontab` (setgid crontab, 2755) writes there. That makes it a documented exception to "no CLI wrapping" (like `smartctl`, `traceroute`); the CLI also signals cron and syntax-checks. Another user's crontab (`user` != caller) needs root: `crontab -u <user>`.
- **Grant, per-user rules** (owner's design). One mechanism for every case, the caller's own account included (matched by name or glob like any other):
  ```yaml
  cron/manage:
    allowed: true
    users:
      deploy:   {view: true, edit: true}
      www-data: {view: true}
      "*":      {view: true}     # everyone else read-only; root and system accounts only if named
  ```
  `view` allows `action: list`, `edit` allows `add` and `remove`. A target that matches no entry is refused; an entry with neither flag is a load error; `allowed: true` with no `users:` is rejected in strict mode and refuses everything at runtime (fails closed), like `containers:`. **No `commands:` allowlist** (owner decision): `edit` therefore means an agent can schedule **any command as that user** - a delayed remote-execution primitive, since mcpd has no arbitrary-command tool today. The docs and the tool description must say so, and the reference config grants it narrowly.
- **Only mcpd's own entries are changed:** every entry `add` creates carries a marker (`# mcpd:<id>`); `remove` takes that id and refuses foreign lines; `list` shows the whole crontab with each line marked mcpd-managed or not (the `view` rule covers everything in that user's crontab).
- **Validation:** schedule = five valid fields (ranges, lists, steps, month/day names) or `@hourly|@daily|@weekly|@monthly|@yearly`; a command with `%` (cron turns it into a newline) is rejected or escaped; empty/oversized command rejected; no newlines in any field.
- **Audit:** every `add` and `remove` is an `audit=true` log line with user, target, id and the full entry.

### Open questions (defaults proposed, the owner decides)

| Question | Proposed default |
|---|---|
| `@reboot` allowed? | no |
| Minimum interval | reject schedules firing more than every 5 minutes |
| Jobs per crontab from mcpd | at most 20 |
| Concurrent edits (read-modify-write race on `crontab -`) | last writer wins, documented; the marker keeps foreign lines intact |
| Root's crontab | only if `root` is named in `users:` |

## Acceptance criteria

- [ ] Decisions in the table above settled (or accepted as proposed), recorded in this ticket.
- [ ] `cron/manage` implemented (`internal/tools/cron/manage/`), registered in `cmd/mcpd/main.go`, `internal/rpc/tools.go` (schema, description per GUIDELINES.md §9, annotations once FR-024 lands, router), with the per-user `users:` rules in the grant parser (`internal/config/sudo.go`) and strict-mode validation.
- [ ] Unprivileged call edits only the caller's crontab; a target other than the caller needs root and a matching `users:` rule; unmatched target, missing `view`/`edit` flag, foreign-line removal and invalid schedules are refused with clear errors.
- [ ] Audit log lines for add/remove.
- [ ] `crontab` documented as a CLI exception in ARCHITECTURE.md; README, docs page with live output, overview, `linuxctl` verbs (`get cron`, `create cron`, `delete cron`) and man page; `configs/mcp-sudo.yaml` (`privileged` block, plus the `mcp-sudo.md` page); `check_docs.sh` and `check_readmes.sh` pass.
- [ ] Deployed and checked live on local k8s, VPS 9091 and 9092.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Schedule validator | Go unit test: valid/invalid 5-field and `@` forms, ranges, steps, names, newlines, `%` | accepts/rejects as specified | | |
| T2 | Rules matching | Go unit test: name, glob, no match, `view` only, `edit` only, root named vs `*`, empty `users:` | refusals and allows as specified | | |
| T3 | Marker handling | Go unit test with a fake `crontab` binary: add, list, remove; foreign lines untouched | only marked lines change | | |
| T4 | Strict config | `ParseSudoConfig` strict: `allowed: true` without `users:`, an entry with no flags | load error; non-strict fails closed | | |
| T5 | Live, unprivileged | VPS 9092 as a throwaway user: add `echo hi >> /tmp/mcpd-cron-test`, list, wait one minute, remove | file written once, crontab clean after | | |
| T6 | Live, privileged target | VPS: root grant, `user: <throwaway>` allowed/refused per rules; foreign crontab entry survives | as specified | | |
| T7 | Boundaries | live: unmatched user, `@reboot`, interval below the minimum, removing a foreign line | each refused with a clear error | | |
| T8 | Docs/build | `check_docs.sh`, `check_readmes.sh`, `GOOS=linux go build ./...`, `go test ./...` | pass | | |

## Comments

- 2026-09-29 - filed. Design settled in chat with the owner: caller's own crontab by default; privileged targets through per-user `view`/`edit` rules in one `cron/manage` grant; no `commands:` allowlist. Everything else in the table above is a proposed default. Live tests use a throwaway account only, never root's crontab and never the amnezia container.
