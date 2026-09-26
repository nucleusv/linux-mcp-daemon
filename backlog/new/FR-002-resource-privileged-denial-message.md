# FR-002 Resources and `describe`: say "not authorized" when root is refused, instead of "permission denied"

- **Created:** 2026-09-26, found while capturing output for the Habr article
- **Related:** ARCHITECTURE.md (resource templates compute their own `isPrivileged` via `SudoConfig.CanReadResourceAsRoot`), `internal/rpc/resources.go`, `cmd/linuxctl` (`describe`, `resource`), FR-001

## Description

A user without a root grant asks for root in three ways; only one of them says that root was refused:

| Command | Output |
|---|---|
| `linuxctl tool files/read --path /root/.bashrc --privileged true` | `user mcp is not authorized to run files/read on path /root/.bashrc as root` - clear |
| `linuxctl resource file:///root/.bashrc --privileged true` | `failed to open file /root/.bashrc: open /root/.bashrc: permission denied (Code: -32603)` |
| `linuxctl describe files /root/.bashrc --privileged true` | every section `(unavailable: ... permission denied)` |

For resources the privilege is decided by the `resources:` grant, not by the caller's `--privileged` flag. Without a grant the read silently runs as the user and fails on file permissions. The caller asked for root and can't tell whether mcpd refused it (policy) or the kernel did (file mode) - the two need different fixes (a grant vs. a chmod). The tool path already gets this right, and logs `WARN tool call denied`; the resource path logs nothing of the kind.

Captured live on the VPS (package v0.3.4, user `mcp` with only a `daemon/reload-config` grant), 2026-09-26.

## Acceptance criteria

- [ ] A resource read with `privileged: true` by a user without the matching `resources:` grant fails with the same kind of message as tools: `user <u> is not authorized to read <uri> as root`, and is logged as `WARN ... denied`.
- [ ] `describe` with `--privileged true` shows that message once (not a `permission denied` per section).
- [ ] A resource read *without* `privileged` behaves as today (runs as the user; kernel errors pass through).
- [ ] A user *with* the grant reads the file as root, as today.
- [ ] `docs/website/docs/configuration/mcp-sudo.md` explains how `privileged` works for resources.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Refused root on a resource | Go unit test on the resource handler: `privileged: true`, no grant | "not authorized" error, no worker spawned | | |
| T2 | No flag, no grant | Go unit test | Runs as the user, kernel error passes through | | |
| T3 | Granted | Go unit test: `resources: file://: ["/root"]` + tool grant | Reads as root | | |
| T4 | CLI `resource` | `tests/test_linuxctl.sh`, unprivileged token, `resource file:///root/.bashrc --privileged true` | "not authorized", non-zero exit | | |
| T5 | CLI `describe` | same, `describe files /root/.bashrc --privileged true` | one "not authorized" line | | |
| T6 | Denial is logged | Live on VPS 9091, then `journalctl -u mcpd` | `WARN ... denied` line for the resource read | | |
| T7 | All deployments | Live on VPS 9092 and local k8s | Same behavior | | |

## Comments

- 2026-09-26 - created. Evidence (VPS, v0.3.4, user `mcp`):
  ```
  $ linuxctl tool files/read --path /root/.bashrc --privileged true
  user mcp is not authorized to run files/read on path /root/.bashrc as root
  $ linuxctl resource file:///root/.bashrc --privileged true
  Error: failed to open file /root/.bashrc: open /root/.bashrc: permission denied (Code: -32603)
  $ linuxctl describe files /root/.bashrc --privileged true
  === stat ===
  (unavailable: failed to stat file /root/.bashrc: stat /root/.bashrc: permission denied)
  ```
