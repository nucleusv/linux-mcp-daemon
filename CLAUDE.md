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
10. Run `bash scripts/check_docs.sh` — it fails if any registered tool lacks a docs page, an overview entry or a `privileged` grant, or if any of those lists a tool that no longer exists. The release workflow runs it too.

## Naming rules (GUIDELINES.md has full detail)
- Tool names: `<group>/<command>` (`files/list`, not `list_files`). No intermediate verb directories (`get/`, `read/`).
- Go package names avoid hyphens/keywords (`cpu/load-average` → package `loadaverage`).
- CLI (`linuxctl <group> <command>`) maps directly to `<group>/<command>` — no fuzzy matching.

## Known stale spots (don't trust blindly)
- `plan/*.md` roadmap docs use pre-rename tool names (e.g. `network/list-interfaces`, `system/get-os-release`) that don't match the current `<group>/<command>` convention. Verify actual tool names against `internal/rpc/tools.go` / `cmd/mcpd/main.go`, not `plan/`.
- `go.mod` module path is still `github.com/nucleusv/linux-mcp-daemon-by-antigravity` even though the GitHub remote has since been renamed twice (first to `linux-mcp-daemon-by-claude`, now to `linux-mcp-daemon` - see `git remote -v`). Renaming the module path means touching every import in the codebase — don't do it opportunistically as a drive-by change.
- The local working directory, Docker image tag, and Kubernetes namespace/deployment/service names (`k8s/*.yaml`, `scripts/build.sh`, `scripts/deploy.sh`, the `Dockerfile`) still say `linux-mcp-daemon-by-claude` - this is a separate, independent naming choice from the GitHub remote and hasn't been renamed to match it. Don't rename these opportunistically either; it would mean tearing down and recreating the live namespace.

## Testing
- Go unit tests adjacent to code (`*_test.go`).
- Integration tests in `tests/` (`test_mcp.sh`, `test_linuxctl.sh`) go through the **MCP JSON-RPC interface**, not raw HTTP/curl against the server.
