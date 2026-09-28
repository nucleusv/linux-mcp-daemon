# FR-019 Release v0.4.0 (Docker tools)

- **Created:** 2026-09-29, by the owner ("what will be in new release, release notes ready?")
- **Related:** FR-011, FR-012, FR-013 (Docker, in review), FR-015, FR-016, FR-018 (linuxctl fixes), CLAUDE.md "Releasing vX.Y.Z", FR-008 (previous release ticket)

## Description

Ship what is on main since v0.3.5: the Docker tool group and the `linuxctl` fixes. The version is a draft: a whole new tool group reads as a minor bump, so 0.4.0; if the owner prefers 0.3.6, rename the notes and the section.

## Acceptance criteria

- [x] Release notes drafted: `docs/release-notes/v0.4.0.md` and a `## 0.4.0` section at the top of `docs/website/docs/release-notes.md`.
- [ ] Owner confirms the version number and the notes.
- [ ] Open tickets in scope settled: FR-015, FR-016, FR-018 through Definition of Done (or left out of the notes); FR-011/012/013 accepted or left in review.
- [ ] Docs current since v0.3.5 (`git log v0.3.5..HEAD`): `check_docs.sh`, `check_readmes.sh` pass; docs site builds.
- [ ] Version bumped by hand: install commands in `docs/website/docs/installation.md`, `"version"` in `server.json`.
- [ ] Deployed and checked live: local k8s, VPS systemd (9091), VPS Docker (9092) - the Docker image and 9091 build must carry the tagged commit.
- [ ] Pushed; /next/ shows the new notes.
- [ ] Tag, release workflow green, assets, docs root, /v0.4.0/, versions.json.
- [ ] MCP Registry (`mcp-publisher publish`) shows the new version active; Glama release.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Docs | `check_docs.sh`, `check_readmes.sh`, Docusaurus build | pass, no broken links | | |
| T2 | Local k8s | `scripts/deploy.sh`; a docker read and a linuxctl call | new build answers | | |
| T3 | VPS 9091 | new binary; `mcpd --version`; docker read as fr011 | tagged commit | | |
| T4 | VPS 9092 | new image; docker manage/exec on throwaway containers only | works; amnezia-awg2 untouched | | |
| T5 | /next/ | curl the release-notes page | 200, 0.4.0 section | | |
| T6 | Release | `gh release view`, workflow status | green, all assets | | |
| T7 | Published docs | root, /v0.4.0/, versions.json | current = v0.4.0 | | |
| T8 | Registry / Glama | registry API, Glama releases | 0.4.0 active | | |

## Comments

- 2026-09-29 - created; notes drafted from `git log v0.3.5..HEAD` and the FR-011/012/013 tickets (chores such as the wiki CI added and removed again are left out). Nothing bumped or tagged yet; the notes claim only behaviour checked live on the VPS stands during FR-015/016/018, plus the Docker features as recorded in their review tickets. The Docusaurus build of the new section was not run yet.
