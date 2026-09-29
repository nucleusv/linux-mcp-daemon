# FR-022 Release v0.4.1 (every tool listed to every user)

- **Created:** 2026-09-29, by the owner ("and updates to docs, release notes?")
- **Related:** FR-020 (the change), FR-017 (test script), FR-019 (0.4.0 release), CLAUDE.md "Releasing vX.Y.Z"

## Description

Ship what is on main since v0.4.0: FR-020 (every tool listed to every user, an ungranted call is an error naming the grant) and the FR-017 test-script rewrite. The version is a draft: the listing change is user-visible, so the owner may prefer 0.5.0; rename the notes and the section if so.

## Acceptance criteria

- [x] Release notes drafted: `docs/release-notes/v0.4.1.md` and a `## 0.4.1` section at the top of `docs/website/docs/release-notes.md`.
- [ ] Owner confirms the version number and the notes.
- [ ] FR-020 and FR-017 accepted (review) or left out of the notes.
- [ ] Docs current since v0.4.0: `check_docs.sh`, `check_readmes.sh`, docs build.
- [ ] Version bumped by hand: install commands in `docs/website/docs/installation.md`, `"version"` in `server.json`.
- [ ] Deployed and checked live at the tagged commit: local k8s, VPS 9091, VPS 9092 (they run a 0.4.1-dev build of f3601be now).
- [ ] Pushed; tag; release workflow green; assets, docs root, /vX/, versions.json; registry (`mcp-publisher publish`); Glama.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Docs | `check_docs.sh`, `check_readmes.sh`, Docusaurus build | pass | | |
| T2 | Deployments | all three stands on the tagged build | `mcpd --version` = the tag; tests/test_linuxctl.sh passes | | |
| T3 | Release | `gh release view`, workflow, docs, registry | all present | | |

## Comments

- 2026-09-29 - created with the drafted notes. Also updated CLAUDE.md's Testing section: test_linuxctl.sh now runs on any stand through env vars. Nothing bumped or tagged yet.
