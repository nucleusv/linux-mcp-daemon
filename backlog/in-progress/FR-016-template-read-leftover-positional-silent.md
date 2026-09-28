# FR-016 A template read silently drops extra positional words instead of warning

- **Created:** 2026-09-29, found while verifying FR-015
- **Related:** FR-015 (`get <group> <keyword> <name>` resolving to templates), `cmd/linuxctl/aggregates.go` (`runDescribe`, `buildDescribeURIs`), `cmd/linuxctl/resolver.go` (`fillTemplate`)

## Description

A tool call that gets more positional words than it needs prints a warning: `mapPositionalArgs` returns the leftover, and `main.go` checks it and prints `Warning: N extra argument(s) ignored: ...`. A template read (`describe <group> <name>`, and since FR-015 also `get <group> <keyword> <name>`) has no equivalent check - `fillTemplate` only consumes as many positional words as the URI template has `{...}` placeholders and returns just the filled URI string, never how many words it actually used. Anything left over is silently discarded.

```text
$ linuxctl get docker network bridge extra-word
{ "Name": "bridge", ... }        # extra-word vanished, no warning, exit 0
```

Not dangerous the way FR-015 was (the right object is still read), but it's the same category of problem: a plainly wrong invocation looks like it succeeded.

## Proposed shape

- `fillTemplate` returns `(string, []string)` - the filled URI and the leftover positional words - instead of just the URI.
- `buildDescribeURIs`'s branches (the simple single-placeholder case, `file:///{path}` which builds two URIs from one filled base, and `process://{pid}/{target}`) propagate the leftover consistently - a Go unit test per branch, since the file/process cases build multiple URIs from one `fillTemplate` call and it's easy to double-count or drop leftover computation there.
- `runDescribe` prints the same `Warning: N extra argument(s) ignored: ...` `main.go`'s tool path already prints, when leftover is non-empty.
- Applies to both `describe <group> <name> <extra>` and `get <group> <keyword> <name> <extra>` (FR-015's new path) - same underlying `runDescribe`/`buildDescribeURIs`, one fix.

## Acceptance criteria

- [ ] `fillTemplate` returns leftover positional words alongside the filled URI.
- [ ] `get docker network bridge extra-word` and `describe docker network bridge extra-word` both warn about `extra-word` and still return the correct data.
- [ ] No regression: `file:///{path}` and `process://{pid}/{target}` describes (which build multiple URIs per call) still work and don't double-warn or warn on the words they do consume.
- [ ] A call with exactly the right number of positional words for its template prints no warning.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Simple template, extra word | Go unit test on `fillTemplate`/`runDescribe` | leftover = `["extra-word"]`, warning printed | 2026-09-29 | pass - `TestBuildDescribeURIsLeftover` "simple extra" |
| T2 | Simple template, exact word count | Go unit test | leftover empty, no warning | 2026-09-29 | pass - "simple exact" |
| T3 | `file:///{path}` describe, extra word | Go unit test | both `/stat` and `/type` reads still happen, one warning, not two | 2026-09-29 | pass - "file extra, no double warn" (2 URIs, 1 leftover) |
| T4 | `process://{pid}/{target}` describe | Go unit test | unaffected - this template already takes an optional target keyword, not just positional filler | 2026-09-29 | pass - "process pid only"/"process extra" |
| T5 | Live | VPS 9091 or 9092 | `get docker network bridge extra-word` and `describe docker network bridge extra-word` both warn | 2026-09-29 | pass - 9091 and 9092 @ 89.125.210.117, commit ab6c4a6, see comments |
| T6 | Docs/tests | `check_docs.sh`, `go test ./...` | pass | 2026-09-29 | pass - `go test -count=1 ./...` all ok natively on the VPS (golang:1.26, linux/amd64) at b375caa; `check_docs.sh` passes |

## Comments

- 2026-09-29 - filed while verifying FR-015: `get docker network bridge extra-word` returns the right object but never mentions the dropped `extra-word`, unlike the equivalent tool-call path. Deliberately not fixed as part of FR-015 - it touches `describe`'s shared code for every group (Case C, which FR-015 left untouched), not just `get`'s new template path, and bundling it in risked a second undertested change on top of a fix that had already gone through two rounds of live-caught bugs.
- 2026-09-29 - moved to in-progress. `fillTemplate` now returns `(uri, leftover)`; `buildDescribeURIs` returns `(uris, extra)` for every branch (file and process build several URIs from one fill and count leftover once); `runDescribe` prints the same `Warning: N extra argument(s) ignored: ...` as the tool path. `aggregates_test.go` `TestBuildDescribeURIsLeftover` covers T1-T4 (9 cases), `go test -count=1 ./cmd/linuxctl/` ok, `GOOS=linux go build ./...` ok, `check_docs.sh` passes. Still open: T5 live run on the VPS stands (needs a linuxctl rebuild + deploy, not done yet).
- 2026-09-29 - committed (481bc95) and deployed `linuxctl` built from HEAD ab6c4a6 to the VPS (systemd stand, 9091; user fr011). Live output:

  ```text
  $ linuxctl get docker network bridge extra-word
  Warning: 1 extra argument(s) ignored: extra-word     # bridge JSON on stdout
  $ linuxctl describe docker network bridge extra-word
  Warning: 1 extra argument(s) ignored: extra-word
  $ linuxctl describe files /etc/hosts x
  Warning: 1 extra argument(s) ignored: x              # one warning, not two
  $ linuxctl get docker network bridge                 # exact count: no warning
  $ linuxctl describe processes 1                      # no warning
  ```

  Not yet: T6 (`go test ./...` is Linux-only on the full tree; only `./cmd/linuxctl/` was run, green) and the 9092 Docker stand (its `privileged` token needs rotating). Staying in in-progress.
- 2026-09-29 - 9092 (Docker stand, user privileged, token rotated, mcpd-docker restarted): `get docker network bridge extra-word` and `describe docker network bridge extra-word` each print `Warning: 1 extra argument(s) ignored: extra-word` with the bridge JSON on stdout, `describe files /etc/hostname x` warns once, `get docker networkz` is still `Error: no read target ...` exit 1, `get docker <TAB>` offers container/containers/exec/image/images/logs/network/networks/volume/volumes. T1-T6 all pass on both stands; what remains is the Definition of Done in backlog/README.md - the owner decides on moving to review.

