# FR-022 Release v0.4.1 (every tool listed to every user)

- **Created:** 2026-09-29, by the owner ("and updates to docs, release notes?")
- **Related:** FR-020 (the change), FR-017 (test script), FR-019 (0.4.0 release), CLAUDE.md "Releasing vX.Y.Z"

## Description

Ship what is on main since v0.4.0: FR-005 (SSE keepalive, added at the owner's request after the notes were drafted), FR-020 (every tool listed to every user, an ungranted call is an error naming the grant) and the FR-017 test-script rewrite. The version is a draft: the listing change is user-visible, so the owner may prefer 0.5.0; rename the notes and the section if so.

## Acceptance criteria

- [x] Release notes drafted: `docs/release-notes/v0.4.1.md` and a `## 0.4.1` section at the top of `docs/website/docs/release-notes.md`.
- [x] Owner confirms the version number and the notes. ("yes 0.4.1")
- [ ] FR-020 and FR-017 accepted (review) or left out of the notes.
- [x] Docs current since v0.4.0: `check_docs.sh`, `check_readmes.sh`, docs build.
- [x] Version bumped by hand: install commands in `docs/website/docs/installation.md`, `"version"` in `server.json`.
- [ ] Deployed and checked live at the tagged commit: local k8s, VPS 9091, VPS 9092 (they run a 0.4.1-dev build of f3601be now).
- [ ] Pushed; tag; release workflow green; assets, docs root, /vX/, versions.json; registry (`mcp-publisher publish`); Glama.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Docs | `check_docs.sh`, `check_readmes.sh`, Docusaurus build | pass | 2026-09-29 | pass - all three; 0.4.1 section present in the built release-notes page |
| T2 | Deployments | all three stands on the tagged build | `mcpd --version` = the tag; tests/test_linuxctl.sh passes | | |
| T3 | Release | `gh release view`, workflow, docs, registry | all present | | |

## Comments

- 2026-09-29 - created with the drafted notes. Also updated CLAUDE.md's Testing section: test_linuxctl.sh now runs on any stand through env vars. Nothing bumped or tagged yet.
- 2026-09-29 - owner: "yes 0.4.1". Bumped server.json and the install commands to 0.4.1, docs build green; tagging v0.4.1 on the bump commit next. FR-020 and FR-017 stay in review (the owner closes them).
- 2026-09-29 - the owner asked to include FR-005. v0.4.1 had already been tagged (e97b155) and its workflow had finished green, so the tag on GitHub does NOT contain FR-005 (8422780). Notes updated for it; waiting for the owner's decision: re-cut v0.4.1 (delete the release and tag, re-tag at the new commit, re-run the workflow - rewrites a published release, GHCR 0.4.1 and /v0.4.1/ docs) or ship it as 0.4.2.

