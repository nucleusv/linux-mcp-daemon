# FR-004 Emptying (truncating) a file must not read it into memory

- **Created:** 2026-09-26, found while preparing FR-003
- **Related:** `internal/tools/files/update/update.go`, FR-003

## Description

The only way to empty a file through mcpd today is `files/update` replacing all lines with an empty string. It writes in place (same inode - correct for a log a service keeps open), but first does `io.ReadAll` of the whole file and splits it into lines. Emptying a multi-GB log on a small host (the test VPS has 2 GB RAM) can exhaust memory - exactly in the "disk is full because of a log" incident where an agent needs it most.

Add a way to truncate without reading: e.g. `files/update` with `truncate: true` (or a size to truncate to), done with `ftruncate` on the opened descriptor, keeping the inode, owner and mode, and respecting `paths` and `_no_follow` like the rest of `files/update`.

## Acceptance criteria

- [ ] A file can be emptied (and optionally truncated to N bytes) without reading its content; memory use doesn't depend on file size.
- [ ] Same inode, owner and mode afterwards; a process appending with O_APPEND keeps writing to it.
- [ ] `paths` grants and symlink protection apply exactly as for other `files/update` calls; the call is audit-logged.
- [ ] Schema description tells agents to truncate logs instead of deleting them.
- [ ] Docs page `mcp-api/tools/files/update.md` updated with live output.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Truncate | Go unit test on a temp file | size 0, same inode/mode | | |
| T2 | Big file, low memory | Go test: 1 GB sparse file, measure heap | heap growth well under file size | | |
| T3 | Writer keeps writing | Go test: O_APPEND writer goroutine | new lines land at offset 0 after truncate | | |
| T4 | Grant boundary | Integration: user with `paths: [/var/log/x.log]` truncates `/var/log/y.log` | not authorized | | |
| T5 | Symlink | Integration: path through a symlink with restricted `paths` | refused | | |
| T6 | Live | VPS 9091/9092, local k8s | as above, audit line in log | | |

## Comments

- 2026-09-26 - created. Kept the FR-003 demo log at ~116 MB so emptying it through today's `files/update` stays safe on the 2 GB VPS.
