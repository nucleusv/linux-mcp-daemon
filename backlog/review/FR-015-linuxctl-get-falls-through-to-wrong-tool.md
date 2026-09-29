# FR-015 linuxctl `get <group> <keyword>` silently runs the wrong tool instead of a resource template

- **Created:** 2026-09-27, found while testing FR-013 (out of scope there, recorded then)
- **Related:** `cmd/linuxctl/resolver.go`'s `Resolve` Case B (`get`) and the `matchToolExactKeyword`/`matchTemplateByKeyword`/`findResourceByTarget`/`matchBareTool` helpers it now composes, `findTemplateByTarget` (still Case C/`describe`-only), FR-011 (`container://`, `image://`, `volume://`), FR-013 (`docker-network://`)

## Description

`get` is documented as *the* read verb. For a resource **template**, it isn't - it silently runs a different tool instead, with the keyword swallowed as a positional argument and only a warning.

```text
$ linuxctl get docker network appnet
Warning: 2 extra argument(s) ignored: network, appnet
<the full docker/containers listing>
```

Same for `get docker volume app-data` and `get docker container web-1 status`: all three run `docker/containers`. The user asked about one network, got every container, exit status 0.

Two independent defects combine:

1. **Case B never consults `reg.Templates`.** At `resolver.go:314-322`, `get` tries `matchToolByLinuxctlVerb` then `findResourceByTarget` - the static resources - and stops. `findTemplateByTarget` exists and is called only by `describe` (Case C). So no `get` can ever resolve to a template, however unambiguous the keyword is.
2. **The keyword fall-through is silent.** When the target keyword matches no granted tool verb in the group, resolution does not fail - it falls through to the group's bare-reachable tool and passes the unmatched words along as positionals, where `mapPositionalArgs` drops them with a `Warning:` on stderr. Under `-s` (or any agent that reads stdout only) even the warning is gone, and the result looks like a successful answer to a question that was never asked.

The second defect is the dangerous one and it is not docker-specific: any group with a bare-reachable read tool will answer the wrong question rather than refuse.

## Proposed shape

- Case B tries templates too, between the tool match and the static-resource match - i.e. `get docker network appnet` resolves to `docker-network://appnet/inspect`, `get docker volume app-data` to `volume://app-data/inspect`, `get docker container web-1 status` to `container://web-1/status`. A template's `linuxctl_verb`/keyword metadata already exists (it is what `describe` matches on).
- An unmatched target keyword is an **error**, not a warning: `no read target "network" in group "docker" - try: linuxctl explain docker`. A keyword that does match is still allowed to pass the rest along as positionals, as today.
- Keep `linuxctl resource <uri>` working unchanged - it is the explicit form and the one the docs show.

Whether a template-backed `get` should print the raw document (like `resource`) or a formatted view is a decision for the ticket, not a requirement: raw is the lazy answer and matches `get`'s "same format as the many-result case" promise least badly.

## Acceptance criteria

- [x] `get <group> <command-name>` names the group's bare-reachable tool instead of warning: `get docker containers` == `get docker`.
- [x] `get <group> <keyword> <name> [args]` resolves to a matching resource template when one exists in that group.
- [x] `get docker network appnet`, `get docker volume app-data`, `get docker image nginx` and `get docker container web-1 status` each read their own template, not `docker/containers`.
- [x] A target keyword that matches no tool verb, resource and template in the group is an error with a non-zero exit status - never a silent fall-through to a different tool.
- [x] `Warning: N extra argument(s) ignored` remains only for genuinely extra arguments after a *successful* match. (Met by FR-016 - originally recorded as **not fully met** - see the 2026-09-29 comment: a genuinely extra positional after a *template* match (`get docker network bridge extra-word`) is silently dropped, no warning. Pre-existing in `runDescribe`/`buildDescribeURIs` (used by `describe` too, untouched here) - `get`'s new template path just inherits it. The dangerous half of this criterion (never answering the wrong question) is fixed; this narrower gap is cosmetic, not incorrect.
- [x] No regression for the bare forms: `get docker`, `get processes`, `get files list /var/log`, `get disks partitions vda`, and `describe <group> <name>` behave exactly as before.
- [x] `get network interfaces` still reads the host resource `network://interfaces`, and does not become ambiguous with a Docker network named `interfaces`.
- [x] Docs: `linuxctl/command-reference.md` and the man page show the new `get <group> <keyword> <name>` forms where they exist.
- [x] Definition of Done (backlog/README.md) - (was blocked on the unchecked criterion above; not moving to `review/` until it's resolved or the owner accepts it as a known, separate follow-up.

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Template resolution | Go unit test on `Resolve` with a fixture registry | `get docker network appnet` → `template_read docker-network://appnet/inspect` | 2026-09-29, macOS | ✅ `TestResolveGetMatchesTemplateByKeyword` PASS |
| T2 | All four docker templates | Go unit test, table-driven | network/volume/image/container each resolve to their own URI, `container://web-1/status` keeps the view arg | 2026-09-29, macOS | ✅ same test, table-driven over all four |
| T3 | Unknown keyword is an error | Go unit test | non-nil error naming the group, no `tool_call` returned | 2026-09-29, macOS | ✅ `TestResolveGetUnmatchedKeywordIsError` PASS |
| T4 | No fall-through in any group | live sweep: `get <group> definitely-not-a-target` for every group the target lists | every group errors; none runs another tool | 2026-09-29, VPS 9092 as privileged | ✅ 14 groups, all exit 1: 12 answer `no read target "definitely-not-a-target" in group "<g>"` (auth cpu daemon disks docker kernel logs memory network services system users), `files` -> `path must be absolute`, `processes` -> `invalid arguments ... pid of type int` (the word is taken as data by the bare tool, an error not a fall-through). Replaces the unit-test sweep this row planned |
| T5 | No regression | Go unit test + live: the bare forms and `describe` listed in the criteria | unchanged output | 2026-09-29, macOS + VPS 89.125.210.117 | ✅ `TestResolveGetAcceptsDefaultToolCommandName` (now with a realistic `files/read` schema - it was passing against a schema-less fixture, which `canConsumePositional` would have broken); live `get docker containers --pattern 'fr011*'` unaffected on both 9091 and 9092 |
| T6 | Host vs docker networking | live: `get network interfaces` and `get docker network bridge` | the host resource and the docker template respectively | 2026-09-29, VPS 9091 and 9092 | ✅ both correct on both stands; regression test `TestResolveGetHostNetworkNotAmbiguousWithDockerNetwork` extended with the real colliding template after the live sweep caught the first version of the fix getting this backwards (see comments) |
| T7 | Live sweep | VPS 9092 as `privileged` | all forms above resolve; real output captured | 2026-09-29, VPS 89.125.210.117:9092 | ✅ output in comments below |
| T8 | Docs | `check_docs.sh`, `check_readmes.sh` | pass | 2026-09-29, macOS | ✅ both pass after `command-reference.md` and the man page were updated |

## Comments

- 2026-09-27 - found while testing FR-013 and left alone there: FR-013 documents `linuxctl resource docker-network://<name>/inspect`, which works, so the ticket needed no workaround. Filed separately because the silent fall-through predates FR-011/FR-013 and affects every group, not just `docker`.
- 2026-09-27 - `new/` → `in-progress/`. Owner reported a third instance of the same fall-through: `linuxctl get docker containers` printed `Warning: 1 extra argument(s) ignored: containers` and then the listing. Not docker-specific either - every group whose bare-reachable tool is named after a word a user would naturally type was affected: `get cpu list`, `get users list`, `get disks list`, `get processes list`, `get files read <path>`. Fixed at the one shared point, `matchToolByLinuxctlVerb`'s group-default fallback (`cmd/linuxctl/resolver.go`): the fallback now also consumes a leading positional equal to the chosen tool's own command name (the segment after `/` in its tool name). Trade-off recorded in the code comment - a positional *value* literally equal to that word (a file called `read`) needs the explicit flag form. Test `TestResolveGetAcceptsDefaultToolCommandName` in `cmd/linuxctl/completion_test.go` covers both forms of each shape plus the no-regression cases.

  Verified live against the FR-011 Docker-in-Docker daemon (`http://localhost:9093`, user `privileged`), after `bash scripts/build-cli.sh`:

  ```text
  $ linuxctl get docker containers          # was: Warning: 1 extra argument(s) ignored: containers
  [running] db-1 (80d3cf9c8b3c)
    Image: redis:alpine
    Status: Up 4 hours
    Ports: 6379/tcp
  $ linuxctl get cpu list
  CPU Information (Total Processors: 4)
  Vendor ID: 0x61
  $ linuxctl get users list
  root (uid=0 gid=0(root)) home=/root shell=/bin/bash
  $ linuxctl get disks list
  NAME   MAJ:MIN RM        SIZE RO TYPE MOUNTPOINTS
  vda    254:0    0 63999836160  0 disk
  $ linuxctl get processes list
  PID  PPID  USER        STAT  RSS       COMMAND
  1    0     root        S     14917632  /usr/local/bin/mcpd
  $ linuxctl get files read /etc/hostname
  15c5e3787744
  $ linuxctl get files /etc/hostname         # unchanged
  15c5e3787744
  $ linuxctl get files list /etc --output table   # unchanged
  NAME                     TYPE      MODE         MODE_OCTAL   LINKS   OWNER   GROUP  ...
  alternatives             dir       drwxr-xr-x   0755         1       root    root   ...
  ```

  No warning on any of them. `go test ./cmd/linuxctl/` → `ok github.com/nucleusv/linux-mcp-daemon/cmd/linuxctl 0.718s`.

  Not fixed yet, still the rest of this ticket: templates under `get` (criterion 2) and an unmatched keyword being an error rather than a warning (criterion 3) - that is the dangerous half. Docs (criterion 7) land with those, not with this.
- 2026-09-27 - checked the other form the owner asked about, `linuxctl restart docker container web-1`: already correct, no change needed. Case A's enum-verb branch consumes `rest[0]` when it equals the tool's `linuxctl_verb`, so both spellings work: `restart docker container web-1` and `restart docker web-1` each printed `Container web-1 (e36aeba4369a): restart succeeded.`
- 2026-09-29 - the dangerous half, finally: Case B now tries an exact tool keyword, then the static resource, then a template, before ever falling back to the group's bare-reachable tool - and if none of those match and the fallback can't take the leftover word as real data either (see `canConsumePositional` below), it's now `Error: no read target "X" in group "Y" - try: linuxctl explain Y` with a non-zero exit, not a silent wrong answer.

  Split the old `matchToolByLinuxctlVerb` (which bundled an exact-keyword check with an *unconditional* bare-tool fallback - the actual bug: it always returned `ok=true` once it found the group's default tool, keyword unconsumed) into `matchToolExactKeyword` and `matchBareTool`, and added `matchTemplateByKeyword`. The hard part wasn't wiring templates in - it was `get files /etc/hosts` and `get processes 1234`, which *also* look like "a positional word that matches no keyword" but must keep working: the bare tool's own schema can genuinely consume them (`path`, `pid`). `canConsumePositional` checks a tool's schema for a `positionalFieldPriority` field or any required field before letting the bare-tool fallback fire for a non-matching word; a word the schema has nowhere to put (`docker/containers` has no positional slot at all) now correctly falls through to the new error instead.

  **Two mistakes made and caught before this was called done, not after:**
  1. Wrote the fix locally, tested it locally (all green), then told the VPS to `git pull` and rebuild - without committing or pushing first. The VPS rebuilt the *old* code; the first live sweep showed every case still broken. Caught by `strings`-checking the deployed binary for the new error text before trusting the live output again (`grep -c 'no read target' /usr/bin/linuxctl` → 0). Committed, pushed, re-pulled, re-verified the string was present *before* copying the binary into place this time.
  2. The first version of the fix checked templates *before* static resources. `network://interfaces` (the plain list, a resource) and `network://interfaces/{name}` (one interface, a template) share the same `linuxctl_verb` "interfaces" - a deliberate kubectl-style get-many/get-one pair. Checking the template first matched it with no name left to fill `{name}` with, and the bare `get network interfaces` broke live (`Error: interface {name} not found`) - a real regression, caught immediately by testing the *whole* sweep live, not just the four cases the ticket names. Fixed by checking resources before templates (docker has zero static resources, so this doesn't reopen the original bug); `dockerFixture` in the test file now includes the colliding template so `TestResolveGetHostNetworkNotAmbiguousWithDockerNetwork` would have caught this itself.

  Verified live on both VPS stands (`89.125.210.117`, commit `01b8860`, native `golang:1.26` build on each, binary string-checked before deploy):

  ```text
  $ linuxctl get docker container fr011-probe status      # was: falls through to docker/containers
  { "created": "2026-09-27T09:07:31...", "state": "running", "uptime": "36h24m26s", ... }

  $ linuxctl get docker network bridge                     # was: falls through
  { "Name": "bridge", "Driver": "bridge", "IPAM": {...}, "Containers": {...}, ... }

  $ linuxctl get docker volume fr011data                   # was: falls through
  { "CreatedAt": "2026-09-27T09:02:37Z", "Driver": "local", "Mountpoint": "/var/lib/docker/volumes/fr011data/_data", "Name": "fr011data" }

  $ linuxctl get docker image nginx:alpine                 # was: falls through
  { "Id": "sha256:df221d...", "RepoTags": ["nginx:alpine"], "Architecture": "amd64", ... }

  $ linuxctl get docker networkz                            # typo - was: falls through silently, exit 0
  Error: no read target "networkz" in group "docker" - try: linuxctl explain docker
  $ echo $?
  1

  $ linuxctl get docker containers --pattern 'fr011*'       # regression check: bare form unaffected
  [running] fr011-probe (810b1f1ee046)
    Image: nginx:alpine
  [running] fr011-logger (71ea9e3754c0)
    Image: busybox

  $ linuxctl get network interfaces                         # regression check: host resource, not the docker template
  [ { "addresses": ["127.0.0.1/8", "::1/128"], ... } ]
  ```

  All of the above reproduced identically on both 9091 (systemd) and 9092 (Docker), and `go test -count=1 ./cmd/linuxctl/...` passed natively on the VPS (x86_64) before either build. `check_docs.sh` and `check_readmes.sh` both pass after updating `linuxctl/command-reference.md` and the man page with the new `get docker <keyword> <name> [view]` form.

  Not moving to `review/`: the leftover-positional-after-a-successful-template-match warning (acceptance criterion 5) is a real, if minor, gap - inherited from `describe`'s existing `runDescribe`, not something this change introduced, but still unmet as written. Filed separately as FR-016 rather than bundled in here.
- 2026-09-29 - follow-up found live on 9091: tab completion for `get docker` offered `exec images logs networks volumes` but not `containers`, nor the new `container`/`network`/`volume`/`image` template keywords. `keywordsFor` skipped tools whose `linuxctl_verb` is `get` (the group's bare tool) and never looked at templates for `get`. Fixed in `cmd/linuxctl/completion.go`; `TestKeywordsForGetDockerIncludesBareToolAndTemplates` fails on the old code (`[images]` only) and passes now.
- 2026-09-29 - review pass. The last open criterion (warning on a genuinely extra positional after a template match) was met by FR-016, verified live on 9091/9092/local k8s; T4's live sweep is above; tab completion for `get docker` (follow-up found live) fixed in ab6c4a6 with a test that fails on the old code. Everything shipped in v0.4.0 (linuxctl) and is on all three stands. Moved to review.

