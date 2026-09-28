# FR-018 `linuxctl exec docker <c> -- <cmd>` runs the wrong command: a bare `--` swallows the next word

- **Created:** 2026-09-29, by the owner ("exec . first" - after the exec `--` result in the docker manage/exec check)
- **Related:** `cmd/linuxctl/main.go` (`splitFlagsAndPositional`), `docs/website/docs/mcp-api/tools/docker/exec.md`

## Description

`splitFlagsAndPositional` treats every token starting with `--` as a flag, including a bare `--`: it becomes a flag named `""` and takes the next word as its value. So `linuxctl exec docker fr011-probe -- echo hello` sent the argv `["hello"]` (`echo` vanished) and the container answered `exec: "hello": executable file not found`, exit 127. Plain `exec docker fr011-probe echo hello` was always fine. `--` is the convention for "everything after this is not for the CLI", and it is the only way to pass an argument that looks like a flag (`ls --color`) to a container.

## Acceptance criteria

- [ ] A bare `--` ends flag parsing; every token after it is positional, verbatim (including ones starting with `-` / `--`).
- [ ] `exec docker <c> -- echo hello` prints `hello`, exit code 0; `exec docker <c> -- ls --color` passes `--color` to `ls`.
- [ ] Flags before `--` still work; no change for calls without `--`.
- [ ] Docs mention it on the exec page.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | `--` ends flags, words kept | `TestSplitFlagsBareDashDashEndsFlags` (fails on the old code: `[web hello]`) | pass | | |
| T2 | Live exec after `--` | VPS 9092, `exec docker fr011-probe -- echo hello` | `hello`, exit 0 | | |
| T3 | Live flag-looking arg | VPS 9092, `exec docker fr011-probe -- ls --color /etc/hostname` | ls output, exit 0 | | |
| T4 | Regression | `exec docker fr011-probe echo hello`, `get processes --limit 2` | unchanged | | |
| T5 | Docs, build | `go test ./cmd/linuxctl/`, `GOOS=linux go build ./...`, `check_docs.sh` | pass | | |

## Comments

- 2026-09-29 - created and in-progress: found while checking docker/manage and docker/exec on 9092; the old client sent `["hello"]` for `-- echo hello`. Fix is a 4-line early exit in the parser plus the test.
