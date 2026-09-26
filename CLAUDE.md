# CLAUDE.md

Agent context for `linux-mcp-daemon`. Full detail lives in [ARCHITECTURE.md](ARCHITECTURE.md) (tool/resource notes, known gotchas) and [GUIDELINES.md](GUIDELINES.md) (naming, code structure, checklists) — read those before non-trivial changes. This file is the condensed, always-loaded version.

## What this is
Go daemon bridging AI agents to a Linux host via MCP (HTTP/SSE + JSON-RPC, port 9091 — changed from the default 9090 to avoid colliding with a sibling deployment; see `configs/daemon.yaml`). Zero-dependency, kernel-first: parse `/proc`, `/sys`, DBus natively instead of wrapping CLI tools, unless a native implementation is impractical (see ARCHITECTURE.md for documented exceptions like `smartctl`, `traceroute`).

This is a **Linux-only** binary (raw `syscall.Utsname`, `syscall.Sysinfo_t`, etc.) — `go build ./...` will fail natively on macOS. Verify with `GOOS=linux go build ./...` instead.

## Architecture
- `cmd/mcpd/main.go` is dual-mode: `mcpd worker <tool> <json-args>` executes one tool and exits; default mode runs the master HTTP/SSE daemon.
- `internal/worker/spawner.go` re-execs the binary in worker mode with `syscall.Credential{Uid: targetUID}` — this is the actual privilege isolation. **The master daemon loop must never read `/proc`/`/sys` or exec external binaries directly.**
- `internal/rpc/tools.go` / `resources.go` — JSON-RPC schema registry (what `tools/list` returns).
- `internal/config/sudo.go` + `configs/mcp-sudo.yaml` — per-user, per-tool root authorization.
- `internal/tools/<group>/<command>/` — one package per tool, entrypoint `func X([]byte) (string, error)`.

## Tasks go through backlog/
Every idea or task from the owner becomes a ticket `backlog/<status>/FR-NNN-name.md` (description, acceptance criteria, a test set written up front, dated comments) before work starts - rules and template in `backlog/README.md`. The folder is the status (`new`, `in-progress`, `blocked`, `review`, `closed`, `rejected`); change it with `git mv` plus a dated comment. Each test's run is recorded in the ticket with date, target and result, with the real output as evidence. Move to `review/` only with every criterion met and every test passing; never move anything to `closed/` - only the owner closes a ticket.

## Adding or changing a tool — do all of these
1. Implement under `internal/tools/<group>/<command>/<command>.go`.
2. Register in the `handlers` map in `cmd/mcpd/main.go` (worker mode dispatch).
3. Register the schema in `internal/rpc/tools.go` (`HandleToolsList`), with a rich `description` and `tools_group`.
4. Add to the `tools/call` router in `internal/rpc/tools.go` (`HandleToolsCall`).
5. Write `internal/tools/<group>/<command>/README.md` — verify with `bash scripts/check_readmes.sh`.
6. If it needs root, add it to `configs/mcp-sudo.yaml` and note it in `docs/website/docs/configuration/mcp-sudo.md`.
7. **Always** add the new tool (or resource) to the `privileged` user block in `configs/mcp-sudo.yaml`, regardless of whether it needs root — that block is intentionally a complete reference granting every tool/resource this daemon exposes, and it's the fastest way to notice a registry change was missed elsewhere: if `privileged` doesn't list something, either the something is stale or the update was incomplete.
8. Add/update `docs/website/docs/mcp-api/tools/<group>/<command>.md`. Note: despite ARCHITECTURE.md mentioning `scripts/generate_docs.py`, **that script does not currently exist** — write the page by hand, following an existing page's format (see `docs/website/docs/mcp-api/tools/services/manage.md`), with output captured live; Docusaurus auto-generates the sidebar from the folder structure.
9. Add the tool to the hand-written list in `docs/website/docs/mcp-api/overview.md` (and to `linuxctl/command-reference.md` / `docs/man/linuxctl.1` if it adds a verb). `tools/list` is generated from the schema and can't drift, but these lists can.
10. Run `bash scripts/check_docs.sh` — it checks every tool, resource and resource template in code against its docs page, the overview list and the reference `privileged` grants (including the second grant a privileged template read needs: the internal worker tool the template spawns), and flags anything listed that no longer exists. The release workflow runs it too. Same for adding a resource or template.
11. Changed how mcpd or linuxctl is *operated* (a new/renamed `linuxctl` verb or flag, an `mcpd` flag, a `daemon.yaml` key, install/package behavior)? Update "Operating mcpd" below in the same commit - it's the always-loaded cheat sheet, and a stale one leads to hand-rolled workarounds.

## Operating mcpd - use the project's own commands
Before giving any ops instruction (install, users, tokens, grants, TLS, reload), use what already exists - never a `journalctl | grep` / hand-edited-YAML workaround:
- mcpd itself: `mcpd [--config-dir DIR]` (default `configs/` relative to the working dir; systemd unit: `WorkingDirectory=/etc/mcpd`), `mcpd --version`, `mcpd worker <tool> <json>` (internal - spawned per call, never run by hand). Configs: `daemon.yaml` (listen, TLS, logging, `worker.containerized`), `users.yaml`, `mcp-sudo.yaml`; live reload via the `daemon/reload-config` tool. It logs its version, listen address and TLS fingerprint at startup.
- Users & grants: `linuxctl create|delete|update mcpd user <name> [--grant TOOL,...] [--set-token V] [--config-path DIR]`, `linuxctl list mcpd users`, `linuxctl describe mcpd user <name>` - they edit `users.yaml`/`mcp-sudo.yaml` atomically and reload the daemon.
- Config by hand: `linuxctl edit mcpd config sudo|users|daemon` (validates, then reloads); apply: `linuxctl reload daemon`.
- TLS: `linuxctl describe mcpd tls --config-path DIR` prints the fingerprint and a ready `export MCP_SERVER=… MCP_TLS_FINGERPRINT=…` line. As root on the mcpd host no fingerprint is needed - linuxctl trusts the cert file directly.
- Client env: `MCP_SERVER`, `MCP_TOKEN`, `MCP_TLS_FINGERPRINT` (or `MCP_CA_CERT`); shell completion: `linuxctl completion bash|zsh`.
- Install: `scripts/install.sh` (`--uninstall` keeps configs), `.deb`/`.rpm` (postinstall prints the next steps; `apt purge` deletes `/etc/mcpd`). Details: `docs/website/docs/installation.md`, `docs/website/docs/linuxctl/`.

## Naming rules (GUIDELINES.md has full detail)
- Tool names: `<group>/<command>` (`files/list`, not `list_files`). No intermediate verb directories (`get/`, `read/`).
- Go package names avoid hyphens/keywords (`cpu/load-average` → package `loadaverage`).
- CLI (`linuxctl <group> <command>`) maps directly to `<group>/<command>` — no fuzzy matching.

## Known stale spots (don't trust blindly)
- `plan/*.md` roadmap docs use pre-rename tool names (e.g. `network/list-interfaces`, `system/get-os-release`) that don't match the current `<group>/<command>` convention. Verify actual tool names against `internal/rpc/tools.go` / `cmd/mcpd/main.go`, not `plan/`.
- The local working directory, Docker image tag, and Kubernetes namespace/deployment/service names (`k8s/*.yaml`, `scripts/build.sh`, `scripts/deploy.sh`, the `Dockerfile`) still say `linux-mcp-daemon-by-claude` - this is a separate, independent naming choice from the GitHub remote and hasn't been renamed to match it. Don't rename these opportunistically either; it would mean tearing down and recreating the live namespace.

## Testing
- Go unit tests adjacent to code (`*_test.go`).
- Integration tests in `tests/` (`test_mcp.sh`, `test_linuxctl.sh`) go through the **MCP JSON-RPC interface**, not raw HTTP/curl against the server.
