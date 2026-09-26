# FR-005 SSE stream: send keepalives so idle clients don't drop the session

- **Created:** 2026-09-26, found during FR-003
- **Related:** `cmd/mcpd/http.go` (`handleSSE`), FR-003

## Description

`handleSSE` writes to the stream only when there is a response to send. While the agent is thinking, or waiting for the human's approval, the stream is silent. `mcp-remote` (the bridge Claude Desktop uses) treats 300 s of silence as a dead connection (`Body Timeout Error`), reconnects and gets a new session (`Re-established session (none) after server expiry`); requests in flight on the old session are lost. In FR-003 this stopped the agent's work in Claude Desktop after the diagnosis step: the owner saw the tasks stopped, and mcpd logged no tool calls after 18:29 although the client kept POSTing.

Fix: send an SSE comment line (`: ping\n\n`) every N seconds (e.g. 15-30 s, configurable in `daemon.yaml`) from the same goroutine that writes events, so writes never interleave. Clients ignore comment lines by spec.

Also check: a config reload (`daemon/reload-config`, `linuxctl edit|create|delete mcpd …`) closes every open SSE session (`session.Done`). Decide whether sessions of users whose grants didn't change should survive a reload, and document the behavior either way.

## Acceptance criteria

- [ ] An idle SSE stream receives a keepalive comment at a fixed interval; the interval is configurable and documented.
- [ ] Keepalives and events never interleave within a line (single writer).
- [ ] A client idle for longer than 300 s (mcp-remote's default body timeout) keeps its session and can call tools afterwards.
- [ ] Reload behavior for open sessions decided and documented (daemon docs).
- [ ] "Operating mcpd" in CLAUDE.md updated if a `daemon.yaml` key is added.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Keepalive sent | Go test: `httptest` SSE client, short interval | `: ping` lines arrive while idle | | |
| T2 | No interleaving | Go test: events and keepalives concurrently | every frame well-formed | | |
| T3 | Long idle | Live: Claude Desktop via mcp-remote, 10 min idle, then a tool call | call succeeds on the same session, no `Body Timeout Error` in the Desktop log | | |
| T4 | Reload | Live: open session, `linuxctl edit mcpd config sudo` with an unrelated change | behaves as decided in the criteria | | |
| T5 | All deployments | VPS 9091, VPS 9092, local k8s | T3 passes | | |

## Comments

- 2026-09-26 - created. Evidence: Claude Desktop log `[72662] ... Body Timeout Error ... Remote SSE stream reconnected ... Re-established session (none) after server expiry`; mcpd log: last `tool call user=agent` at 18:29:02, then only `POST /message?session_id=… status=202` from new sessions. `handleSSE` loop in `cmd/mcpd/http.go` has no ticker.
