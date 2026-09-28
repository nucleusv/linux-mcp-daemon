# FR-018 `linuxctl exec docker <c> -- <cmd>` runs the wrong command: a bare `--` swallows the next word

- **Created:** 2026-09-29, by the owner ("exec . first" - after the exec `--` result in the docker manage/exec check)
- **Related:** `cmd/linuxctl/main.go` (`splitFlagsAndPositional`), `docs/website/docs/mcp-api/tools/docker/exec.md`

## Description

`splitFlagsAndPositional` treats every token starting with `--` as a flag, including a bare `--`: it becomes a flag named `""` and takes the next word as its value. So `linuxctl exec docker fr011-probe -- echo hello` sent the argv `["hello"]` (`echo` vanished) and the container answered `exec: "hello": executable file not found`, exit 127. Plain `exec docker fr011-probe echo hello` was always fine. `--` is the convention for "everything after this is not for the CLI", and it is the only way to pass an argument that looks like a flag (`ls --color`) to a container.

## Acceptance criteria

- [x] A bare `--` ends flag parsing; every token after it is positional, verbatim (including ones starting with `-` / `--`).
- [x] `exec docker <c> -- echo hello` prints `hello`, exit code 0; `exec docker <c> -- ls --color` passes `--color` to `ls`.
- [x] Flags before `--` still work; no change for calls without `--`.
- [x] Docs mention it on the exec page.
- [x] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | `--` ends flags, words kept | `TestSplitFlagsBareDashDashEndsFlags` (fails on the old code: `[web hello]`) | pass | 2026-09-29 | pass - fails on old code (`[web hello]`), passes now |
| T2 | Live exec after `--` | VPS 9092, `exec docker fr011-probe -- echo hello` | `hello`, exit 0 | 2026-09-29 | pass - VPS 9092, linuxctl from a14f1c5: `hello`, exit 0 |
| T3 | Live flag-looking arg | VPS 9092, `exec docker fr011-probe -- ls --color /etc/hostname` | ls output, exit 0 | 2026-09-29 | pass - `ls --color` got the flag (ANSI codes in the output), exit 0 |
| T4 | Regression | `exec docker fr011-probe echo hello`, `get processes --limit 2` | unchanged | 2026-09-29 | pass - `exec ... echo hello` and `get processes --limit 2` unchanged |
| T5 | Docs, build | `go test ./cmd/linuxctl/`, `GOOS=linux go build ./...`, `check_docs.sh` | pass | 2026-09-29 | pass - `go test ./cmd/linuxctl/` ok, `GOOS=linux go build ./...` ok, `check_docs.sh` passes |

## Comments

- 2026-09-29 - created and in-progress: found while checking docker/manage and docker/exec on 9092; the old client sent `["hello"]` for `-- echo hello`. Fix is a 4-line early exit in the parser plus the test.
- 2026-09-29 - fixed in a14f1c5 and deployed to the VPS (`/usr/bin/linuxctl`, 9092 as privileged). Live: `exec docker fr011-probe -- echo hello` → `hello`, exit 0; `-- ls --color /etc/hostname` → coloured output, exit 0; `-- sh -c "echo a; echo b"` → `a` `b`; the no-`--` form and `get processes --limit 2` unchanged. Definition of Done left: redeploy of local k8s (the daemon is untouched, only the client changed) - owner decides on review.
- 2026-09-29 - redeployed everywhere the client lives, at a8c4957: local k8s (`scripts/deploy.sh`, pod rolled out; the Mac `executables/linuxctl` rebuilt by it; `describe files /etc/hosts x` warns once), VPS systemd (`/usr/bin/linuxctl`, earlier, a14f1c5) and VPS Docker (`mcpd-docker` image rebuilt on the VPS and only that container recreated with its same flags - host net/pid, privileged, `/` bound read-only at /host, configs mounted, restart unless-stopped). Inside the Docker container: `linuxctl exec docker fr011-probe -- echo hello` → `hello`, exit 0; `-- ls --color /etc/hostname` → coloured output; `get docker network bridge extra-word` → the FR-016 warning. amnezia-awg2 stayed `Up 3 days`, untouched. Correction to an earlier remark of mine: the Docker image *does* ship linuxctl (`/usr/local/bin/linuxctl`), it just was stale. All criteria met - moved to review.

