# FR-008 Release v0.3.5 (stdio mode) and a Release Notes docs page

- **Created:** 2026-09-26, by the owner ("подготовь документацию 0.3.5 … после сделай релиз")
- **Related:** FR-007 (stdio mode), FR-006 (catalogs: registry version, Glama release), CLAUDE.md "Releasing vX.Y.Z"

## Description

Ship what's on main since v0.3.4 - `mcpd stdio` and the notification fix - as v0.3.5, following the release checklist in CLAUDE.md. Add a "Release Notes" page to the docs site; for now it holds only this release (owner: "пока релиз ченджес - что вошло в этот релиз").

## Acceptance criteria

- [x] Docs: `docs/release-notes/v0.3.5.md`; `docs/website/docs/release-notes.md` with a 0.3.5 section; install commands and `server.json` at 0.3.5; checklist in CLAUDE.md mentions the page.
- [x] Deployed and checked live: local k8s, VPS systemd (9091), VPS Docker (9092).
- [x] Pushed; /next/ shows the new pages.
- [x] Tag v0.3.5; release workflow green; assets (archives, .deb/.rpm, checksums, GHCR image) present; docs root, /v0.3.5/ and versions.json updated.
- [x] MCP Registry: `mcp-publisher publish` → 0.3.5 active (with the new description).
- [x] Glama: release created (Auto-Release on the GitHub release, or Make Release).
- [x] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Docs build | local Docusaurus build, check_docs, check_readmes | success, no broken links | 2026-09-26, local | ✅ SUCCESS, checks pass |
| T2 | Local k8s | scripts/deploy.sh; `linuxctl get cpu load-average`; `kubectl exec -i … mcpd stdio --user testuser` | new build answers | 2026-09-26 | ✅ load average; stdio 37 tools + memory/usage |
| T3 | VPS 9091 | new binary; `mcpd --version`; tool call; MCP Inspector → `ssh root@vps mcpd stdio --user agent` | works; stdio over SSH | 2026-09-26 | ✅ 0.3.5-dev active; load average; over SSH 37 tools, Ubuntu 24.04, privileged refused |
| T4 | VPS 9092 | amd64 image built locally, `docker save | ssh docker load`, container recreated with the same flags; stdio call inside | works; AmneziaVPN untouched | 2026-09-26 | ✅ listening :9092 https; os-release via stdio; VPN Up 27 h before/after |
| T5 | /next/ | curl the new pages | 200 | 2026-09-26 | ✅ release-notes, stdio-mode, installation 200; install commands V=0.3.5 |
| T6 | Release | gh release view v0.3.5; workflow status | green, all assets | 2026-09-27 | ✅ release.yml success; 9 assets; GHCR 0.3.5 amd64+arm64 |
| T7 | Published docs | root, /v0.3.5/, versions.json | current = v0.3.5 | 2026-09-27 | ✅ all 200; versions.json current v0.3.5 |
| T8 | Registry | registry API | 0.3.5 active | 2026-09-27 | ✅ 0.3.5 active, isLatest, new description; 0.3.4 kept |
| T9 | Glama | releases page | a release exists | 2026-09-27 | ✅ 0.3.5 via Auto-Release; TDQS A 3.5/5 (37 tools) |

## Comments

- 2026-09-26 - created, in progress. Docs written: release notes (GitHub text + docs page, 0.3.5 only for now), install commands and server.json bumped.
- 2026-09-26 - deployed to all three targets (details in the tests table). VPS 9091 runs a main build over the v0.3.4 package binary (backup /root/mcpd-0.3.4.bak) until the 0.3.5 package is installed. VPS Docker image built for amd64 on the Mac instead of on the 2 GB VPS next to AmneziaVPN.
- 2026-09-27 - released v0.3.5 (tag on 8ed63f4). Registry: the first publish failed with an expired registry JWT; two device-code logins expired before approval; the owner then ran `mcp-publisher login github` themselves and publish succeeded. Glama created release 0.3.5 by itself (Auto-Release). Found on the way: Glama renders a README from its first heading and drops raw `<img>` - README logo and diagram moved to Markdown images below the title (425ff9f, ca2cc3b); both render on Glama now. Moved to review.
