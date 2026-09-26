# FR-009 Docs: plain `linuxctl`, not `./executables/linuxctl`

- **Created:** 2026-09-27, by the owner ("убрать везде ./executables/ - у нас же по нормальным путям всё ставится")
- **Related:** README.md, docs/website/docs/linuxctl/autocompletion.md

## Description

A release install (`install.sh`, `.deb`/`.rpm`) puts `linuxctl` on `PATH`; `./executables/linuxctl` is only where `scripts/build-cli.sh` writes a source build. The README's examples used the build path everywhere, which reads as if that were how to run it.

## Acceptance criteria

- [x] No `./executables/linuxctl` in user-facing docs; examples call `linuxctl`.
- [x] The build-from-source steps say to add `executables/` to `PATH` (the only place it remains), and that a release install already is on `PATH`.
- [x] Docs site builds; check_docs passes.

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Gone | `grep -rn './executables/' README.md docs/website/docs` | nothing | 2026-09-27 | ✅ 0 (19 replaced in README, 1 reworded in autocompletion.md) |
| T2 | Docs build | local Docusaurus build, check_docs | success | 2026-09-27 | ✅ SUCCESS, check_docs passes |

## Comments

- 2026-09-27 - done, to review.
