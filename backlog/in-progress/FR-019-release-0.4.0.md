# FR-019 Release v0.4.0 (Docker tools)

- **Created:** 2026-09-29, by the owner ("what will be in new release, release notes ready?")
- **Related:** FR-011, FR-012, FR-013 (Docker, in review), FR-015, FR-016, FR-018 (linuxctl fixes), CLAUDE.md "Releasing vX.Y.Z", FR-008 (previous release ticket)

## Description

Ship what is on main since v0.3.5: the Docker tool group and the `linuxctl` fixes. The version is a draft: a whole new tool group reads as a minor bump, so 0.4.0; if the owner prefers 0.3.6, rename the notes and the section.

## Acceptance criteria

- [x] Release notes drafted: `docs/release-notes/v0.4.0.md` and a `## 0.4.0` section at the top of `docs/website/docs/release-notes.md`.
- [x] Owner confirms the version number and the notes. ("new version" - taken as go-ahead for 0.4.0)
- [ ] Open tickets in scope settled: FR-015, FR-016, FR-018 through Definition of Done (or left out of the notes); FR-011/012/013 accepted or left in review.
- [ ] Docs current since v0.3.5 (`git log v0.3.5..HEAD`): `check_docs.sh`, `check_readmes.sh` pass; docs site builds.
- [x] Version bumped by hand: install commands in `docs/website/docs/installation.md`, `"version"` in `server.json`.
- [ ] Deployed and checked live: local k8s, VPS systemd (9091), VPS Docker (9092) - the Docker image and 9091 build must carry the tagged commit.
- [x] Pushed; /next/ shows the new notes.
- [x] Tag, release workflow green, assets, docs root, /v0.4.0/, versions.json.
- [x] MCP Registry (`mcp-publisher publish`) shows the new version active; Glama release.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Docs | `check_docs.sh`, `check_readmes.sh`, Docusaurus build | pass, no broken links | 2026-09-29 | pass - all three (build output in a temp dir; new 0.4.0 section present) |
| T2 | Local k8s | `scripts/deploy.sh`; a docker read and a linuxctl call | new build answers | 2026-09-29 | pass - pod rolled out at a8c4957; describe warns on the extra word (redo at the tagged commit) |
| T3 | VPS 9091 | new binary; `mcpd --version`; docker read as fr011 | tagged commit | | |
| T4 | VPS 9092 | new image; docker manage/exec on throwaway containers only | works; amnezia-awg2 untouched | | |
| T5 | /next/ | curl the release-notes page | 200, 0.4.0 section | 2026-09-29 | pass - /next/ 200, release-notes page contains 0.4.0 |
| T6 | Release | `gh release view`, workflow status | green, all assets | 2026-09-29 | pass - run 36500838778: test, image, binaries all green; 9 assets (2 rpm, 2 deb, 2 linux tar.gz, 2 darwin linuxctl, checksums); GHCR 0.4.0 manifest 200 |
| T7 | Published docs | root, /v0.4.0/, versions.json | current = v0.4.0 | 2026-09-29 | pass - root, /v0.4.0/, /v0.4.0/release-notes/ 200; versions.json current v0.4.0 |
| T8 | Registry / Glama | registry API, Glama releases | 0.4.0 active | 2026-09-29 | registry pass (0.4.0 active, latest); Glama release not yet confirmed - page 200, Auto-Release to be checked |

## Comments

- 2026-09-29 - created; notes drafted from `git log v0.3.5..HEAD` and the FR-011/012/013 tickets (chores such as the wiki CI added and removed again are left out). Nothing bumped or tagged yet; the notes claim only behaviour checked live on the VPS stands during FR-015/016/018, plus the Docker features as recorded in their review tickets. The Docusaurus build of the new section was not run yet.
- 2026-09-29 - docs build (`docusaurus build --out-dir <tmp>`) succeeds with the 0.4.0 section; `server.json` and the install commands bumped to 0.4.0. Local k8s and the VPS Docker stand (image rebuilt, only mcpd-docker recreated) run a8c4957. Waiting: the owner's go for the irreversible steps (tag push, registry publish) - and FR-016 moves to closed/ after the release, as the owner asked.
- 2026-09-29 - MCP Registry published ahead of the tag, at the owner's explicit request ("lets do mcp-publisher publish"; the owner ran `mcp-publisher login github` themselves): `mcp-publisher publish` → `io.github.nucleusv/linux-mcp-daemon` 0.4.0 active, isLatest true; 0.3.4 and 0.3.5 kept. `server.json` has only a remote URL, no packages, so the entry does not depend on release assets. Still to do, in order: tag v0.4.0 and push it (release.yml), verify the assets, docs root, /v0.4.0/ and versions.json, Glama release, then close FR-011/012/013/016 per the owner.
- 2026-09-29 - tagged v0.4.0 on ff12422 (annotated, pushed over 443) and released. Verified: workflow green (test 44s, image 3m, binaries 1m18s), 9 assets, docs root + /v0.4.0/ + versions.json current v0.4.0, GHCR 0.4.0. FR-011, FR-012, FR-013 and FR-016 moved to closed/ as the owner asked. Left open here: Glama release confirmation, T3 (VPS 9091 mcpd still runs the pre-tag main build dev-9a0a31b - the daemon code is the same, the tagged release binary is not installed there) and T4 (VPS Docker image reports version=dev) - a reinstall of the 0.4.0 package on 9091 would make `mcpd --version` say 0.4.0; owner decides.

