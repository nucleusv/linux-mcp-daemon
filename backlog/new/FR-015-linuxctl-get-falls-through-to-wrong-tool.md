# FR-015 linuxctl `get <group> <keyword>` silently runs the wrong tool instead of a resource template

- **Created:** 2026-09-27, found while testing FR-013 (out of scope there, recorded then)
- **Related:** `cmd/linuxctl/resolver.go:314-322` (Case B), `cmd/linuxctl/resolver.go:326` (Case C, the only place `findTemplateByTarget` is used), FR-011 (`container://`, `image://`, `volume://`), FR-013 (`docker-network://`)

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

- [ ] `get <group> <keyword> <name> [args]` resolves to a matching resource template when one exists in that group.
- [ ] `get docker network appnet`, `get docker volume app-data`, `get docker image nginx` and `get docker container web-1 status` each read their own template, not `docker/containers`.
- [ ] A target keyword that matches no tool verb, resource and template in the group is an error with a non-zero exit status - never a silent fall-through to a different tool.
- [ ] `Warning: N extra argument(s) ignored` remains only for genuinely extra arguments after a *successful* match.
- [ ] No regression for the bare forms: `get docker`, `get processes`, `get files list /var/log`, `get disks partitions vda`, and `describe <group> <name>` behave exactly as before.
- [ ] `get network interfaces` still reads the host resource `network://interfaces`, and does not become ambiguous with a Docker network named `interfaces`.
- [ ] Docs: `linuxctl/command-reference.md` and the man page show the new `get <group> <keyword> <name>` forms where they exist.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Template resolution | Go unit test on `Resolve` with a fixture registry | `get docker network appnet` → `template_read docker-network://appnet/inspect` | | |
| T2 | All four docker templates | Go unit test, table-driven | network/volume/image/container each resolve to their own URI, `container://web-1/status` keeps the view arg | | |
| T3 | Unknown keyword is an error | Go unit test | non-nil error naming the group, no `tool_call` returned | | |
| T4 | No fall-through in any group | Go unit test over every group in a full fixture registry: `get <group> definitely-not-a-target` | every one errors; none returns a `tool_call` | | |
| T5 | No regression | Go unit test + live: the bare forms and `describe` listed in the criteria | unchanged output | | |
| T6 | Host vs docker networking | live: `get network interfaces` and `get docker network bridge` | the host resource and the docker template respectively | | |
| T7 | Live sweep | VPS 9092 as `privileged` | all forms above resolve; real output captured | | |
| T8 | Docs | `check_docs.sh`, `check_readmes.sh` | pass | | |

## Comments

- 2026-09-27 - found while testing FR-013 and left alone there: FR-013 documents `linuxctl resource docker-network://<name>/inspect`, which works, so the ticket needed no workaround. Filed separately because the silent fall-through predates FR-011/FR-013 and affects every group, not just `docker`.
