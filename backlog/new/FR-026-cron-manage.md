# FR-026 Crontabs: `get crontabs` / `update crontabs` / `edit crontabs`, own crontab free, other users' behind sudo and per-user rules

- **Created:** 2026-09-29, by the owner (design agreed in chat)
- **Parent:** [FR-023](FR-023-linux-utilities-coverage-gap-analysis.md) - survey of everyday Linux utilities (this is a subtask of it)
- **Related:** FR-025 (`timers/list`), `internal/tools/docker/manage` and its grants (the injection pattern for daemon-supplied values), `internal/tools/kernel/system-control` (read when no value, write when a value is given), FR-024 (description checklist, GUIDELINES.md §9 - description and annotations for the new tool)

## Description

One tool, **`cron/manage`**, shown in `linuxctl` as the group **`crontabs`** (`tools_group: "crontabs"`, so `get crontabs`, not `get cron manage`). There is no `action` parameter: the tool **reads when the `content` key is absent and writes when it is present** (an explicit empty `content` clears the crontab), the way `kernel/system-control` does.

| Call | Result |
|---|---|
| `linuxctl get crontabs` | the caller's **own** crontab, headed by the user name; "test_user has no crontab." is not an error |
| `linuxctl get crontabs --privileged true` | **list** of the users that have a crontab (name, lines, modified) - only accounts the caller has a named `view` rule for **that actually have a crontab**; own crontab included if it exists |
| `linuxctl get crontabs test_user --privileged true` | that user's crontab |
| `linuxctl update crontabs [test_user] [--privileged true] [--if_match HASH] --content "..."` | replace the whole crontab; `--content` is required (an `update` never turns into a read) |
| `linuxctl edit crontabs [test_user] [--privileged true]` | `crontab -e`-style: read (remember the hash), `$EDITOR` on a temp copy, write back with `if_match`; client-side only, built from the same read and write |
| `linuxctl describe crontabs test_user [--privileged true]` | the resource template `crontab://{user}`: the crontab as plain text (read-only) |

### Whose crontab, and who decides

- **Own crontab (no sudo, no grant).** The daemon decides the target: the authenticated mcpd user's pinned OS account. The per-call worker already runs with that UID, so `crontab -l` and `crontab -` (no `-u`) act on exactly that user; no user name travels from the caller. A `user` equal to the caller's own account behaves like omitting it.
- **Another user's crontab (sudo).** `privileged: true` (root worker, a grant), and the target must match a per-user rule in the grant. The master validates the name (charset, account exists), checks the rule, and injects the verified name in a reserved argument (delete-then-set, as `_containers` is for docker); the root worker runs `crontab -u <verified name>`. A caller-supplied reserved key is dropped.
- **Per-user rules - named accounts only, no wildcards** (owner decision):
  ```yaml
  cron/manage:
    allowed: true
    users:
      test_user: {view: true, edit: true}   # read and replace this user's crontab
      www-data:  {view: true}               # read only
  ```
  Each key is one account name. A key with a glob character (`*`, `?`, `[`) or an invalid account name is a load error, so there is no precedence to reason about and `root` is covered only when named. `view` allows the read, the list and the template; `edit` allows `update`/`edit` and implies `view` (so `{edit: true}` is valid). An entry with neither flag is a load error; `allowed: true` without `users:` is rejected in strict mode and fails closed at runtime (like `containers:`). An account with no entry is refused. **No command allowlist** (owner decision): `edit` means the agent can schedule any command as that user - the docs and the description say so.
- **Own account:** always view and edit, no sudo, no rule; the `users:` list and `privileged` are ignored when the target is the caller.
- **The list** is built natively by root reading the spool directory (`/var/spool/cron/crontabs` on Debian/Ubuntu, `/var/spool/cron` on RHEL): file name = user, line count and mtime from the file. Only reading and writing one crontab uses the `crontab` command (setgid crontab: a user cannot write their own spool file), a documented CLI exception like `smartctl`.
- **Host limits stay:** `/etc/cron.allow` and `cron.deny` are enforced by `crontab`; its message is returned as is. No `crontab` command or no spool directory -> a clear error, not a crash.
- **Write safety:** `crontab` rejects a file it cannot parse and keeps the old one. Optional `if_match` (sha256 of the crontab as read) refuses the write if it changed since; without it the last writer wins. Every write is an audit line: caller, target, size, line count and a content hash (never the content, it may hold secrets).

## Open (proposed defaults, owner confirms with "defaults")

1. `if_match` included (above).
2. (settled) no wildcards: only named accounts, so root is covered only when named.
3. An empty `content` is allowed and clears the crontab.
4. `linuxctl` positional priority list gets `user` at its end so `get crontabs test_user` fills the `user` parameter (check `get processes`, `get users` and the resolver tests stay unchanged).

## Acceptance criteria

- [ ] `cron/manage` (`internal/tools/cron/manage/`): read, list (privileged, no user), write; registered in `cmd/mcpd/main.go` and `internal/rpc/tools.go` (schema with `tools_group: "crontabs"`, description per GUIDELINES.md §9, router) and in `internal/rpc/annotations.go` (destructive, non-idempotent-safe values per the analysis); `users:` rules parsed and validated in `internal/config/sudo.go`.
- [ ] Resource template `crontab://{user}` in `internal/rpc/resources.go`, same permission check and worker as the tool.
- [ ] `linuxctl`: `get`, `update`, `edit`, `describe` for `crontabs` as in the table; `user` added to the positional priority list without changing other commands.
- [ ] Own crontab works unprivileged with no grant; another user's needs root and a matching rule; unmatched user, missing `view`/`edit`, invalid crontab (old one kept), `if_match` mismatch, `update` without `--content` give clear errors; the list shows only viewable users.
- [ ] Audit line for every write.
- [ ] **Risks documented** (owner: "write the risks down ... link with a warning from the edit/update page"): the risk table is written in `docs/website/docs/configuration/permissions-and-risks.md` (section "Crontabs", marked planned until release) and pointed to from `mcp-api/overview.md`; when the tool ships, remove the "planned" banners, and the `cron/manage` docs page and the `linuxctl` `update`/`edit` `crontabs` entries in the command reference each open with a `:::danger` box - "Writing a crontab schedules commands as that user, and survives the end of the session and the revocation of the token. Read the [crontab risks](../../../configuration/permissions-and-risks#crontabs) first." - linking there. Also keep the table in step with the tool's real behaviour (name validation, `cron.deny` check, size cap, `if_match`).
- [ ] `crontab` listed as a CLI exception in ARCHITECTURE.md; README, docs page with live output, overview, `linuxctl` command reference and man page, `configs/mcp-sudo.yaml` (`privileged` block) and `mcp-sudo.md`; `check_docs.sh` and `check_readmes.sh` pass.
- [ ] Deployed and checked live on local k8s, VPS 9091 and VPS 9092.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Rules matching | Go unit test: named account, unlisted account, view-only, edit-only (implies view), root named/unnamed, own account ignores rules, empty `users:` | allow/refuse as specified | | |
| T2 | Strict config | `ParseSudoConfig` strict: no `users:`, an entry with no flags, a key with `*`/`?`/`[` or an invalid name | load error; non-strict fails closed | | |
| T3 | Target injection | Go unit test: caller-supplied reserved keys dropped; own-account `user` treated as own; another user needs root | as specified | | |
| T4 | Read/write/list | Go unit test with a fake `crontab` binary and a fake spool dir: read, list (filtered by view), write, invalid file keeps old, `if_match` mismatch | as specified | | |
| T5 | linuxctl grammar | Go unit tests on `Resolve`: `get crontabs`, `get crontabs test_user`, `update crontabs ...`, `describe crontabs ...`; existing `get processes <pid>` unchanged | resolves as in the table | | |
| T6 | Live, own crontab | VPS 9092 as an unprivileged user with an account on the host: `update`, `get`, back to empty | round trip exact, crontab left empty | | |
| T7 | Live, other users | VPS: `get crontabs --privileged true` lists only viewable users; read/write allowed and refused targets | as specified | | |
| T8 | Docs/build | `check_docs.sh`, `check_readmes.sh`, `GOOS=linux go build ./...`, `go test ./...` | pass | | |

## Comments

- 2026-09-29 - filed, cut down, then rewritten to the agreed design: group `crontabs`, no `action` parameter, own crontab free, privileged listing of users that have a crontab, per-user view/edit rules for others, `edit` verb, template `crontab://{user}`. The example account is `test_user` (illustrative; the live tests use a real throwaway account, never root's crontab and never the amnezia container).
- 2026-09-29 - owner asked for the risks to be written into the docs: section "Crontabs" (planned) added to permissions-and-risks.md and a pointer in the overview; the warning box on the tool and command pages is an acceptance criterion above and lands with the implementation.
- 2026-09-29 - owner: remove the asterisk from the cron rules; a privileged listing shows only the crontabs of accounts with `view`. Design now: named accounts only (no globs, no precedence, root only when named); entry flags default to false, `edit` implies `view`.
