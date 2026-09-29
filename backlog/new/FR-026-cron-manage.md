# FR-026 cron/manage: read a user's crontab and write it whole, with per-user view/edit rules

- **Created:** 2026-09-29, by the owner ("edit = we can fully write the cron file")
- **Parent:** [FR-023](FR-023-linux-utilities-coverage-gap-analysis.md) - survey of everyday Linux utilities (this is subtask of it)
- **Related:** FR-023 (research, section (c)), FR-025 (`timers/list`), `internal/tools/docker/manage` and its `containers:` grant (the grant pattern), FR-024 (description checklist, GUIDELINES.md §9)

## Description

One tool, `cron/manage`, two actions:

- `read` - returns the target user's whole crontab.
- `write` - replaces the target user's whole crontab with the `content` given. No entry parsing, no markers: `crontab` itself rejects a file it cannot parse, and then nothing is replaced.

`user` is optional and defaults to the caller. The caller's own crontab is handled unprivileged, through the `crontab` command (`crontab -l`, `crontab -`) - a user cannot write their own spool file (VPS: `/var/spool/cron/crontabs` is `root:crontab` 1730, `/usr/bin/crontab` is setgid crontab 2755), so this is a documented CLI exception like `smartctl`. Another user's crontab is `crontab -u <user>` as root.

Grant, per-user rules (one mechanism for the caller's own account and for others; matched by name or glob):

```yaml
cron/manage:
  allowed: true
  users:
    deploy:   {view: true, edit: true}
    www-data: {view: true}
    "*":      {view: true}      # everyone else read-only; root and system accounts only if named
```

`view` allows `read`, `edit` allows `write`. No matching entry -> refused; an entry with neither flag -> load error; `allowed: true` without `users:` -> rejected in strict mode, fails closed at runtime (like `containers:`). `edit` means the agent can schedule **any command as that user** - the docs and the description say so plainly. Every `write` is an audit line: caller, target user, size, line count and a hash of the content (not the content, which may hold secrets).

## Acceptance criteria

- [ ] `cron/manage` (`internal/tools/cron/manage/`) with `read` and `write`, registered in `cmd/mcpd/main.go` and `internal/rpc/tools.go` (schema, description per GUIDELINES.md §9, router), `users:` rules parsed and validated in `internal/config/sudo.go`.
- [ ] Own crontab works unprivileged; another user's needs root and a matching rule; an unmatched user, a missing `view`/`edit` flag, and an invalid crontab (rejected by `crontab`, old one kept) give clear errors.
- [ ] Audit line for every `write`.
- [ ] `crontab` listed as a CLI exception in ARCHITECTURE.md; README, docs page with live output, overview, `linuxctl` verbs, `configs/mcp-sudo.yaml` (`privileged` block) and `mcp-sudo.md`; `check_docs.sh` and `check_readmes.sh` pass.
- [ ] Deployed and checked live on local k8s, VPS 9091 and 9092.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Rules matching | Go unit test: name, glob, no match, view-only, edit-only, root named vs `*`, empty `users:` | allow/refuse as specified | | |
| T2 | Strict config | `ParseSudoConfig` strict: no `users:`, an entry with no flags | load error; non-strict fails closed | | |
| T3 | Read/write | Go unit test with a fake `crontab` binary: write then read back; an invalid file keeps the old one | as specified | | |
| T4 | Live, unprivileged | VPS 9092, a throwaway account: write a crontab with one harmless job, read it back, write it empty again | round trip exact, crontab left empty | | |
| T5 | Live, privileged target | VPS: root grant, allowed and refused targets per the rules | as specified | | |
| T6 | Docs/build | `check_docs.sh`, `check_readmes.sh`, `GOOS=linux go build ./...`, `go test ./...` | pass | | |

## Comments

- 2026-09-29 - filed, then cut down at the owner's word ("it is too much", "edit = fully write the cron file"): dropped `cron/list`, entry markers, schedule validation, entry limits and the open-questions table. Live tests use a throwaway account only - never root's crontab, never the amnezia container.
