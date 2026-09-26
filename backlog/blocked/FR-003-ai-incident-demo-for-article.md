# FR-003 Live demo for the Habr article: an AI agent resolves an incident through mcpd only

- **Created:** 2026-09-26, by the owner
- **Related:** the Habr draft (habr.com/ru/article/edit/1086976/), FR-002

## Description

The article explains how mcpd works but never shows an agent actually solving a problem. Stage a real incident on the test VPS and let a real AI agent (Claude Code) diagnose and fix it, then put its actual tool calls and answers into a new article section.

Rules of the demo:
- **Setup and cleanup** (the "world" the incident happens in) are done over SSH as root.
- **Solving** is done **only through MCP**: the agent uses mcpd's tools with the token of a dedicated MCP user `agent` and nothing else - no SSH while solving.
- Only throwaway objects created for the demo are touched. AmneziaVPN (`amnezia-awg2`), sshd and mcpd itself are off limits.

Scenario (changed by the owner 2026-09-26): a chatty `my-service.service` keeps appending DEBUG lines to `/var/log/my-service.log` (~116 MB and growing). The complaint: "disk usage on the server keeps growing, find out why". The agent should find the log, show what fills it, and **ask the owner for approval** to empty it in place (`> /var/log/my-service.log`), not delete it - the service keeps the file open, so deleting would free nothing. Then verify the space came back and the service still writes to the same file (same inode).

The `agent` user's only root grant: `files/update` with `paths: [/var/log/my-service.log]`. Everything else runs as the OS user `agent`.

## Acceptance criteria

- [x] The chatty service, its growing log and the `agent` MCP user with narrow grants exist on the VPS; the agent's token never appears in chat or logs.
- [x] Claude Desktop is connected to mcpd on the VPS over HTTPS as `agent` (via `mcp-remote@0.14.3 --header-file`, cert trusted with `NODE_EXTRA_CA_CERTS`).
- [ ] The agent, using MCP tools only, finds the growing log, asks the owner before touching it, empties it in place and verifies (size, same inode, service still writing).
- [ ] An attempt outside the grant (e.g. reading `/etc/shadow` as root) is refused, and the refusal is shown.
- [ ] mcpd's log shows every call of the session, with the change marked `audit`.
- [ ] A new article section shows complaint → steps (call + trimmed real answer) → result → log lines.
- [ ] Demo objects removed from the VPS afterwards (unit, files, OS user; the MCP user `agent` kept or removed as the owner decides).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Incident is real | SSH: `du -h /var/log/my-service.log`, `stat -c %i`, `systemctl is-active my-service` | ~116 MB and growing, service active | 2026-09-26, VPS | ✅ 116M, inode 40860, active |
| T2 | Agent is MCP-only | The session transcript: every agent action is an `mcp__…` tool call | no SSH/Bash against the VPS while solving | | |
| T3 | Diagnosis | Agent's calls | the log found as the biggest grower, its content shown, the writing service identified | | |
| T4 | Approval | Chat | the agent asks before emptying the file and waits | | |
| T4b | Fix | Agent's call `files/update` with `privileged: true` | file emptied in place | | |
| T5 | Verified | Agent's calls + SSH check after | small size, same inode 40860, new lines keep arriving | | |
| T6 | Boundary | Agent's call outside the grant | "not authorized", logged as denied | | |
| T7 | Audit trail | SSH: `journalctl -u mcpd` for the session | every call listed; `files/update` marked audit | | |
| T8 | Cleanup | SSH | demo unit, files, OS user gone; mcpd, Docker 9092 and AmneziaVPN untouched | | |

## Comments

- 2026-09-26 - created and moved to in-progress: owner agreed on scenario A, setup over SSH, solving only via MCP.
- 2026-09-26 - scenario changed by the owner from a broken config to a growing log that the agent empties after asking. Setup over SSH: `my-service` (python, as `nobody`) appending to `/var/log/my-service.log`, prefilled to 116 MB (kept small on purpose, see FR-004); MCP user `agent` created with `linuxctl create mcpd user` (token only in a root-only file on the VPS), its single root grant set with `linuxctl edit mcpd config sudo`: `files/update` on `/var/log/my-service.log`. The earlier demo-web unit and polkit rule removed. AmneziaVPN checked before/after every step: up.
- 2026-09-26 - owner chose Claude Desktop (screenshots) over Claude Code. Connected through `mcp-remote@0.14.3 --transport sse-only --header-file ~/.config/mcpd/agent.headers` with `NODE_EXTRA_CA_CERTS=~/.config/mcpd/vps.crt`. First attempt failed: the header file still held the placeholder, mcpd answered 401 and mcp-remote fell back to OAuth dynamic client registration, which mcpd doesn't serve (404) - "Server disconnected". After the owner wrote the token into the file: Desktop shows `mcpd-vps-agent: Running`, `tools/list` and `resources/list` answered, mcpd logs `user=agent ... uri=/sse status=200`. Demo state before the dialog: log 123M, inode 40860; AmneziaVPN up.
- 2026-09-26 - diagnosis done in Claude Desktop through MCP only: 20 tool calls (mcpd log 18:27:38-18:29:02), incl. a `disks/usage privileged: true` attempt that mcpd refused (T6 material) and a `logs/journal-control` call the kernel refused (agent isn't in systemd-journal). The owner shared the agent's answer; rendered as two Desktop-style images (`~/Downloads/habr-demo-1-calls.png`, `-2-answer.png`, "оформление воспроизведено" caption needed). Next step (approval + truncate) blocked: the Desktop session dropped after 300 s of SSE silence - see FR-005. `my-service` stopped at the owner's request; log stays at 125M, inode 40860.
- 2026-09-26 - moved to blocked: waiting on FR-005 (SSE keepalive) or a fresh Desktop session for the approval step.
- 2026-09-26 - owner: the article gets a GIF of the dialog (no full video). Habr can't host video files; a GIF up to 8 MB loads like any image. The owner records the Desktop window (Cmd+Shift+5), I convert it.
- 2026-09-26 - owner changed the plan: no GIF, the two rendered images go into the article as screenshots. Uploaded to habrastorage (calls: https://habrastorage.org/webt/b5/26/ff/b526ffcb296f7f612c9e3aa56e2a8133.png, answer: https://habrastorage.org/webt/09/b5/7e/09b57efdc0a15e81be33675695328c14.png) and added a section "Как ИИ разбирает инцидент" before "Обратите внимание" with the caption "оформление воспроизведено, ответ агента сокращён". The approval + truncate step is not in the article.
