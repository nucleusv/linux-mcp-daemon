# FR-001 `linuxctl describe disks <name>`: a readable summary, not raw counters

- **Created:** 2026-09-26, by the owner
- **Related:** `docs/website/docs/linuxctl/grammar.md` (definition of `describe`), `investigations/feature-backlog.md`

## Description

`linuxctl describe disks vda` reads the `disks://{name}/stats` template and prints the raw `/proc/diskstats` counters as JSON (sectors read/written, ms spent on I/O - cumulative since boot). That contradicts the project's own definition of `describe` in the grammar: "rich, aggregated single-object detail ... combines several reads into one report", like `kubectl describe`. Only `describe processes <pid>` does that today.

`describe disks <name>` should print one human-readable report about the disk built from the tools that already exist:

```
Name:        vda
Size:        ...             (disks/list)
Model/Type:  ... / virtio    (disks/list)
Partitions:  vda1 ...        (disks/partitions)
Mounts:      /  ext4  used 45%   (disks/mounts, disks/usage)
I/O:         read ... MB/s, write ... MB/s, util ...%   (disks/performance - measured over an interval)
Health:      ...             (disks/health, when smartctl is available)
```

The raw counters stay reachable through `linuxctl resource disks://vda/stats`.

Also check the other `describe` targets that print a single raw JSON object - `system services <name>`, `network interfaces <name>` - and either bring them to the same standard or record why not.

## Acceptance criteria

- [ ] `linuxctl describe disks <name>` prints a `Key: value` report with size, model/type, partitions, mounts with usage, I/O rates and health (a section that can't be read says why, e.g. "Health: smartctl not installed", instead of failing the whole command).
- [ ] `-o json` / `-o yaml` return the same report as structured data.
- [ ] A disk that doesn't exist gives a clear error.
- [ ] `resource disks://<name>/stats` still returns the raw counters.
- [ ] Decision recorded for `describe system services` and `describe network interfaces` (changed, or a comment why not).
- [ ] `command-reference.md`, `grammar.md` and the man page show the new output, captured live on the VPS.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Report is assembled from the right tools | Go unit test: fake tool responses for list/partitions/mounts/usage/performance/health → formatter | Every section filled from its source, `Key: value` layout | | |
| T2 | One section failing doesn't fail the command | Go unit test: `disks/health` returns an error | Report printed, `Health: <reason>`, exit code 0 | | |
| T3 | Unknown disk | Go unit test + `tests/test_linuxctl.sh`: `describe disks nosuchdisk` | Clear error naming the disk, non-zero exit | | |
| T4 | Structured output | `tests/test_linuxctl.sh`: `describe disks <first disk> -o json` | Valid JSON with the same sections | | |
| T5 | Raw counters still reachable | `tests/test_linuxctl.sh`: `resource disks://<first disk>/stats` | The `/proc/diskstats` fields as today | | |
| T6 | Real host | Live on VPS 9091 (`vda`, virtio, no SMART) | Size, partitions, `/` mount with usage, I/O rates; Health explains smartctl can't read a virtual disk | | |
| T7 | Container deployment | Live on VPS 9092 (Docker) | Same report for the host's disk, not the container's | | |
| T8 | Dev node | Live on local k8s | Report for the LinuxKit VM's disk; missing sections explained | | |
| T9 | Unprivileged user | Live, a user without root grants | Sections that need root say so; the rest is shown | | |
| T10 | Docs match reality | `bash scripts/check_docs.sh`; outputs in `command-reference.md` pasted from T6 | Passes; docs show the live output | | |

## Comments

- 2026-09-26 - Found while collecting output for the Habr article: the owner ran `linuxctl describe disks vda` on the VPS (package v0.3.4) and asked whether `describe` should print that. It matches the current docs, but not the grammar's definition of `describe`.
