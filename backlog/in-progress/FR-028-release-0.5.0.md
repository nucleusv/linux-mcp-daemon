# FR-028 Release v0.5.0 - timers, crontabs, docker/run, Glama-ready tool descriptions

- **Created:** 2026-09-29, by the owner ("release ticket with all tasks included"; docker/run "next release")
- **Related:** FR-022 (0.4.1, released), FR-027 (article, after this release), CLAUDE.md "Releasing vX.Y.Z"

## Description

The next release after v0.4.1. The version is a proposal: new tools (`timers/list`, `crontabs`, `docker/run`) and MCP annotations on every tool, so **0.5.0**; the owner confirms.

## Subtasks

- [FR-025](../review/FR-025-timers-list.md) - `timers/list` - review (built, deployed on all stands)
- [FR-024](FR-024-glama-tool-quality-descriptions-annotations.md) - Glama tool quality: annotations (done), 46 rewritten descriptions (done, second read open), guide (done), 14 follow-up code bugs listed in its section E - in-progress
- [FR-026](../new/FR-026-cron-manage.md) - crontabs: `get|update|edit crontabs`, per-user view/edit rules, worker as the target UID, resource templates - new
- [FR-014](../new/FR-014-docker-run-allowed-images.md) - `docker/run` from allowed images with per-image settings - new (the owner: in this release; security-heavy, built after FR-026)
- Not in this release: FR-023's other proposals (ssh, certs, time, ...), FR-027 (the article, published after it).

## Acceptance criteria

- [ ] Every subtask above is done and reviewed, or explicitly moved out of the release by the owner.
- [ ] FR-024 follow-ups decided: which of the 14 code bugs (`cpu/list topology_only`, docker/exec 300 s limit, curl/ping number-vs-int types, ...) are fixed in this release and which are filed for later.
- [ ] The "second read" of the 46 descriptions done by a reader other than the author (FR-024 T2).
- [ ] Docs current since v0.4.1: tool pages with live output, `configuration/*`, `linuxctl/*`, man page, README, ARCHITECTURE.md (incl. the stale `disks/fdisk`/`network/traceroute` names and the `crontab` CLI exception); every "planned" banner for crontabs removed (README "Crontabs" section, overview note, risks section) and the risk warning box on the crontab tool and command pages; `check_docs.sh`, `check_readmes.sh`, docs build.
- [ ] Release notes written twice: `docs/release-notes/v0.5.0.md` and a `## 0.5.0` section in `docs/website/docs/release-notes.md`; install commands and `server.json` bumped to 0.5.0.
- [ ] Deployed and checked live on local k8s, VPS 9091 (the tagged package) and VPS 9092 (the tagged image); `tests/test_linuxctl.sh` passes on all three; the raw-API sweep (tools, resources, templates) and stdio smoke check pass.
- [ ] Tag, workflow green, assets, docs root and `/v0.5.0/`, `versions.json`; MCP Registry (`mcp-publisher publish`, the owner logs in); Glama release and a re-check of every tool's score (FR-024 T4).
- [ ] Loose ends from 0.4.1 closed: FR-022, FR-019, FR-005 (Claude Desktop / mcp-remote idle test by the owner).
- [ ] After the release: FR-027 (article), and closing of the finished subtasks by the owner.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Docs | `check_docs.sh`, `check_readmes.sh`, Docusaurus build | pass, no broken links | | |
| T2 | All stands | `mcpd --version` = the tag on local k8s, 9091, 9092; suite on each | green | | |
| T3 | API sweep | raw MCP `tools/list`, `resources/list`, templates, a safe call each; stdio smoke | no unexpected errors; every tool annotated | | |
| T4 | Release | `gh release view`, workflow, docs, registry | all present | | |
| T5 | Glama | tool page after the release | every tool grade A, no dimension <= 2 | | |

## Comments

- 2026-09-29 - created with all tasks linked. Order of work: FR-025 (done) -> FR-026 crontabs -> FR-014 docker/run -> FR-024 remainder and the description second read -> docs and notes -> deploy -> tag -> registry. The registry already holds 0.4.1 (published today).
