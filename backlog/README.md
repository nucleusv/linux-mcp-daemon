# Backlog

The task tracker for this project. Every idea or task from the owner becomes a ticket here before any work starts, and the work follows the ticket.

A ticket is one file, `FR-NNN-short-name.md`, and everything about the task lives in it - description, criteria, tests, results, the real output used as evidence, decisions, links to commits. No side files, no scattered notes. **Its folder is its status** - changing status means moving the file with `git mv`, so the history of every move stays in git. The board is just `ls backlog/*/`.

## Statuses

`new/` → `in-progress/` → `review/` → `closed/`, plus `blocked/` and `rejected/`.

| Folder | Meaning |
|---|---|
| `new/` | Recorded, not started. |
| `in-progress/` | Being worked on. |
| `blocked/` | Waiting on something - the latest comment says what. |
| `review/` | Done: every acceptance criterion checked off and every test passed, with evidence; waiting for the owner. |
| `closed/` | The owner accepted it. **Only the owner closes a ticket.** |
| `rejected/` | Decided not to do - the latest comment says why. |

Every move and every decision gets a dated comment in the ticket. A ticket goes to `review/` only when all its criteria are met and all its tests pass; otherwise the unmet ones stay unchecked with a comment.

## Tests

Every ticket gets its test set written *before* work starts, in its "Tests" table: what each case checks, how (a Go unit test, a `tests/*.sh` integration case through the MCP interface, or a live check on a named target), and the expected result. New automated cases go into the repo (`*_test.go`, `tests/`), not just into the ticket. When a case runs, the table gets the date, the target (local k8s, VPS 9091, VPS 9092, a container) and `✅ pass` / `❌ fail`, with the evidence - the real output or the test name - in a comment. A failed case is fixed and re-run; the table keeps the latest result, the comments keep the history.

## Definition of Done

For any code change, on top of a ticket's own criteria (see CLAUDE.md): unit tests pass, `GOOS=linux go build ./...`; docs updated and `bash scripts/check_docs.sh` passes; redeployed to local k8s, VPS systemd (9091) and VPS Docker (9092) and checked live with real output; the "Operating mcpd" section of CLAUDE.md updated if operation changed; committed.

## Ticket template

```markdown
# FR-NNN Title

- **Created:** YYYY-MM-DD, by the owner
- **Related:** files, docs, other tickets

## Description

What and why.

## Acceptance criteria

- [ ] ...
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | ... | ... | ... | | |

## Comments

- YYYY-MM-DD - created: ...
```

IDs are never reused: the next one is one above the highest in any folder.
