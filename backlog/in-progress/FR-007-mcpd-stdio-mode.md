# FR-007 `mcpd stdio`: serve MCP over stdin/stdout, as the invoking user, never root

- **Created:** 2026-09-26, by the owner
- **Related:** FR-006 (Glama needs a server it can start over stdio), `cmd/mcpd/main.go` (modes), `cmd/mcpd/http.go`, `internal/rpc`, `internal/worker/spawner.go`

## Description

mcpd only speaks MCP over the network (HTTP/SSE with a token). Many MCP clients and catalogs start a server as a local process and talk to it over stdin/stdout. Glama's automated checks require exactly that and reject bridges (`mcp-remote` is refused: "The Dockerfile must build and run the server locally, not proxy to an external endpoint").

Add a mode: `mcpd stdio [--config-dir DIR]` reads JSON-RPC from stdin and writes responses to stdout, through the same RPC handlers the HTTP server uses (`tools/list`, `tools/call`, `resources/*`). Logs go to stderr only - stdout carries protocol messages and nothing else.

Security model - it must not become a way around mcp-sudo:
- No token, no MCP user: the caller is the OS user who started the process (`getuid()`); workers run as that user.
- `privileged: true` is always refused in stdio mode, whatever `mcp-sudo.yaml` says - root over stdio would mean root for whoever controls the client's config. If mcpd is started as root, it refuses to start in stdio mode unless an explicit flag says so (to be decided; default: refuse).
- The same per-tool config (`tools:` timeouts, `network:` limits) applies. Calls are logged to stderr in the same format, `transport=stdio`.

When it is useful (goes into the docs page):
1. **The agent's own machine** - a laptop or dev box: diagnose the host the agent runs on with no daemon, no port, no TLS or tokens.
2. **Over SSH, with no open port**: the client runs `ssh user@host mcpd stdio` as the server command. Auth is the existing SSH key, nothing listens on the network, and the agent still gets typed tools instead of a shell. Honest caveat: whoever holds that SSH key can open a shell anyway - this limits the *agent*, not the key.
3. **Inside a container or pod**: `docker exec -i <container> mcpd stdio`, `kubectl exec -i <pod> -- mcpd stdio` - look at a container from the inside, as its own user.
4. **Trying mcpd out** in a minute, before installing the daemon, users and TLS.
5. **Catalogs, CI and MCP Inspector**: Glama's checks, `npx @modelcontextprotocol/inspector mcpd stdio`, integration tests without HTTP.
6. **Air-gapped or locked-down hosts** where opening a port is not allowed.

When not to use it: remote access for several people or agents, per-user identities, root for specific tools, central audit - that's the network daemon.

## Acceptance criteria

- [x] `mcpd stdio` answers `initialize`, `tools/list`, `tools/call`, `resources/list|read` over stdin/stdout; stdout has only JSON-RPC.
- [x] Workers run as the invoking OS user; `privileged: true` is refused with a clear error; started as root → refuses by default (decision recorded).
- [x] Logs go to stderr, marked `session=stdio`.
- [x] Docs: a new page `docs/website/docs/configuration/stdio-mode.md` (use cases above, security model, client config examples for Claude Desktop/Code, SSH, `docker exec`, `kubectl exec`, MCP Inspector); links from installation, the AI-agent guide and `permissions-and-risks`; man page `mcpd.8`; CLAUDE.md "Operating mcpd".
- [x] Glama Dockerfile config switched to `CMD ["mcpd", "stdio"]` (plus the build steps) and Glama's build passes; listing shows in Glama search (closes the rest of FR-006's Glama item).
- [x] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Protocol over stdio | Go test `TestServeStdio`: initialize, tools/list, 2 notifications, blank line, bad JSON, unknown method | valid responses, nothing but JSON-RPC on stdout | 2026-09-26, golang:1.26 container, non-root | ✅ 4 responses, none for notifications, -32700 / -32601 |
| T2 | Runs as caller | live: `mcpd stdio` as user `tester`; `system/os-release`, `files/read /etc/shadow` | works; shadow denied by the kernel | 2026-09-26, container | ✅ os-release OK; `permission denied` |
| T3 | No root | `tools/call files/read /etc/shadow privileged:true`; Go test `TestSpawnWorkerNoRootRefusesPrivileged` (incl. `read_*`, even with a grant) | refused with the stdio message, logged | 2026-09-26, container | ✅ `privileged: true is not available when mcpd serves over stdio…`; stderr `WARN tool call denied` |
| T4 | Started as root | root without `--user`; `--user root`; `--user agent` + `/proc/self/status` | refuse, refuse, worker as agent | 2026-09-26, container | ✅ both refused; worker `Uid: 999` = agent |
| T5 | Client: Claude Desktop | `claude_desktop_config.json` `command: mcpd, args: [stdio]` on a Linux box | tools listed and callable | - | ⏭ not run: no Linux desktop here; the same stdio path is covered by MCP Inspector (T8) and SSH (T6) - owner to decide |
| T6 | Over SSH | MCP Inspector → `ssh root@vps mcpd stdio --user agent` | works, no port opened | 2026-09-26, VPS | ✅ 37 tools, os-release, privileged refused |
| T7 | Container | `docker exec -i stdio-t mcpd stdio --user agent` | tools from inside the container | 2026-09-26, local Docker | ✅ (via T8) |
| T8 | MCP Inspector | `inspector --cli <docker exec wrapper> --method tools/list | resources/list | tools/call memory/usage` | connects, lists tools | 2026-09-26, Mac → container | ✅ 37 tools, 11 resources, memory/usage OK |
| T9 | Glama | Glama admin → Build, CMD `mcp-proxy -- mcpd stdio --user glama` | build and checks pass | 2026-09-26 | ✅ test success (1m 9s); release 0.3.5 |
| T10 | Docs | `check_docs.sh`, `check_readmes.sh`, local Docusaurus build; live /next/ after push | passes, page renders | 2026-09-26, local | ✅ checks pass, build SUCCESS; ⏳ /next/ after push |

## Comments

- 2026-09-26 - created. Trigger: Glama rejected the Dockerfile config that ran mcpd over HTTP inside the container behind `mcp-remote` (tested locally first: image built, `initialize` + `tools/list` returned 37 tools through mcp-proxy). Owner asked for a separate docs page on when stdio mode is useful.
- 2026-09-26 - moved to in-progress: owner asked to implement stdio mode in detail. Protocol version upgrade (2024-11-05 → newer) kept out of this ticket - separate research ticket.
- 2026-09-26 - implemented: `cmd/mcpd/stdio.go` (`mcpd stdio [--user NAME] [--config-dir DIR]`, one writer goroutine, concurrent requests, 64 MiB line limit, optional daemon.yaml for timeouts/logging); `worker.NoRoot` + `worker.ErrNoRoot` checked in `SpawnWorker` (covers tools and resources incl. `read_*`) and early in `HandleToolsCall` so the message isn't the grant-missing one; a non-root mcpd starts workers without switching uid (setgroups needs CAP_SETGID). Also fixed for both transports: notifications (no `id`) are never answered - before, anything but `notifications/initialized` got a `Method not found` with id null. Decided: started as root → `--user NAME` required (needed for Glama/docker exec), `--user root` refused. Logs carry `session=stdio` (not a separate transport= field). Full `go test ./...` passes as non-root in golang:1.26. Docs: `configuration/stdio-mode.md`, links from ai-agent-configuration and permissions-and-risks, `mcpd.8` synopsis + STDIO MODE section, CLAUDE.md cheat sheet.
- 2026-09-27 - shipped in v0.3.5; T6 and T9 pass. T5 not run (no Linux desktop) - left for the owner to accept or ask for. Stays in-progress until the owner decides on T5.
