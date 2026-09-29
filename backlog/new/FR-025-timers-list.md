# FR-025 timers/list: read systemd timers (what runs on a schedule, and when next)

- **Created:** 2026-09-29, by the owner ("yes do timers, will they be resources? are they changeable?")
- **Parent:** [FR-023](FR-023-linux-utilities-coverage-gap-analysis.md) - survey of everyday Linux utilities (this is subtask of it)
- **Related:** FR-023 (ranked proposal, rank 1), `internal/tools/services/list` (same DBus code, mirrors its shape), FR-026 (`cron/manage`), FR-024 (description checklist in GUIDELINES.md §9 applies)

## Description

On modern hosts systemd timers are the real scheduler: the Ubuntu VPS (89.125.210.117) has 18 timers (`apt-daily`, `logrotate`, `fstrim`, `dpkg-db-backup`, `sysstat-collect`, ...) against 0 user crontabs. An agent cannot see any of them today: `services/list` keeps only `.service` units (`internal/tools/services/list/list.go`), and `services/manage` forces a `.service` suffix (`foo.timer` becomes `foo.timer.service`).

New read-only **tool** `timers/list` (a list with filters is a tool, like `services/list`; not a resource). Group `timers`, `linuxctl get timers`. Unprivileged, over the same DBus connection as `services/list` (system bus for a normal user, systemd's private socket as root). Per timer: name, description, `active_state`, `sub_state`, the unit it triggers, next run, last run, the schedule (`OnCalendar=` specs and monotonic `OnBootSec`/`OnUnitActiveSec` values), `Persistent`, randomized delay, last result. Filters: `pattern` (same wildcard rules as services/list), `active_state`, `output_format`. Times as RFC 3339 UTC, and "never" when systemd reports 0.

**Changeable?** Not in this ticket - read-only. Follow-ups, each its own ticket if wanted: (1) let `services/manage` accept `.timer` units so an existing timer can be started, stopped, enabled and disabled (a small change: it now appends `.service` unconditionally); (2) creating a timer (unit files + `daemon-reload`) - bigger and riskier, ranked 12 in FR-023 (`timers/run`, transient timers over DBus `StartTransientUnit`). A single-timer resource template `timer://{name}/status` (like `service://{name}/status`) is possible later; `timers/list --pattern` already covers it.

## Acceptance criteria

- [ ] `timers/list` implemented under `internal/tools/timers/list/`, registered in `cmd/mcpd/main.go` (worker handlers), `internal/rpc/tools.go` (schema with `tools_group: "timers"`, `linuxctl_verb: "get"`, description per GUIDELINES.md §9, and the `tools/call` router).
- [ ] Output (text and `output_format: json`) has the fields above; a timer that has never run and an inactive timer render "never"/empty, not garbage.
- [ ] `README.md` for the package, docs page `docs/website/docs/mcp-api/tools/timers/list.md` with live output, the overview list, `linuxctl` command reference; `configs/mcp-sudo.yaml` `privileged` block lists it; `check_docs.sh` and `check_readmes.sh` pass.
- [ ] Deployed and checked live on local k8s, VPS 9091 and VPS 9092.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Property parsing | Go unit tests on the helpers that turn DBus values into times and schedule strings (calendar and monotonic specs, 0 = never) | correct strings | | |
| T2 | Filters | Go unit test: pattern / active_state on a fixed slice | only matching timers | | |
| T3 | Live, real host | VPS: `linuxctl get timers` and `--output json` | the 18 timers, next/last run plausible against `systemctl list-timers` | | |
| T4 | Unprivileged | as an unprivileged user (testuser, 9092) | works without a grant | | |
| T5 | No systemd | local k8s node (no DBus) | a clear error, not a crash | | |
| T6 | Docs/build | `check_docs.sh`, `check_readmes.sh`, `GOOS=linux go build ./...`, `go test ./...` on the VPS | pass | | |

## Comments

- 2026-09-29 - filed after the owner asked "why is it needed" and "resources? changeable?": needed because timers are invisible today; a tool, not a resource; read-only now (see Description for the write follow-ups).
