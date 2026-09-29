# FR-024 Raise every tool's Glama quality score: rewrite descriptions to the six dimensions, add MCP annotations, and a guide for new tools

- **Created:** 2026-09-29, by the owner ("I have Glama tools verify ... files/create - behavior is yellow and usage guidelines ... analyze what is needed for each tool, put on task to redo, and to Claude as guide for the new tools")
- **Related:** FR-010 (the 8 lowest tools - superseded in scope, left for the owner to close), FR-006 (Glama listing), GUIDELINES.md §9 (the guide, written with this ticket), `internal/rpc/tools.go`, `internal/rpc/docker.go`, `docs/website/docs/mcp-api/tools/`, each tool's `README.md`

## Description

Glama's tool page (https://glama.ai/mcp/servers/nucleusv/linux-mcp-daemon, per-tool detail at `#<tool>`, e.g. `#files/create`) scores each tool 1-5 on six dimensions from the tool's `description` and schema:

| Dimension | Glama's question |
|---|---|
| Behavior | Does the description disclose side effects, auth requirements, rate limits, or destructive behavior? |
| Conciseness | Is it appropriately sized, front-loaded and free of redundancy? |
| Completeness | Given the tool's complexity, does it cover enough for an agent to succeed on the first attempt? |
| Parameters | Does it clarify parameter syntax, constraints, interactions or defaults beyond the schema? |
| Purpose | Does it state what the tool does and how it differs from similar tools? |
| Usage Guidelines | Does it explain when to use it, when not to, or what alternatives exist? |

What the page showed for the release it scored (v0.3.5, 37 tools; a later release adds the 8 `docker/*` tools and more - 46 callable tools now). Only these 20 rows were readable; the rest of the 37 need to be read off the page (T4):

| Tool | Grade | Score | Beh | Con | Cmp | Par | Pur | Use |
|---|---|---|---|---|---|---|---|---|
| auth/sudo-rules | A | 3.8 | 3 | 5 | 4 | 3 | 5 | 3 |
| cpu/list | A | 4.0 | 2 | 5 | 4 | 3 | 5 | 5 |
| cpu/load-average | A | 4.0 | 3 | 5 | 4 | 3 | 5 | 4 |
| disks/free | A | 4.0 | 4 | 5 | 4 | 3 | 4 | 4 |
| disks/health | A | 4.2 | 4 | 5 | 4 | 3 | 5 | 4 |
| disks/list | A | 4.7 | 4 | 5 | 5 | 4 | 5 | 5 |
| disks/mounts | A | 4.2 | 3 | 5 | 4 | 3 | 5 | 5 |
| disks/partitions | A | 3.9 | 3 | 4 | 4 | 3 | 5 | 4 |
| disks/performance | A | 4.2 | 3 | 5 | 4 | 4 | 5 | 4 |
| disks/usage | A | 3.9 | 2 | 5 | 3 | 3 | 5 | 5 |
| files/chmod | A | 4.4 | 4 | 5 | 4 | 4 | 5 | 4 |
| files/chown | A | 4.4 | 5 | 5 | 4 | 3 | 5 | 4 |
| files/create | B | 3.1 | 2 | 5 | 3 | 3 | 4 | 2 |
| files/filetype | A | 4.4 | 4 | 5 | 4 | 3 | 5 | 5 |
| files/find | C | 2.7 | 2 | 3 | 1 | 3 | 4 | 2 |
| files/list | A | 4.0 | 3 | 5 | 4 | 3 | 5 | 4 |
| files/read | B | 3.4 | 3 | 4 | 3 | 3 | 4 | 3 |
| files/update | A | 3.7 | 3 | 5 | 3 | 3 | 5 | 3 |
| kernel/system-control | A | 4.2 | 4 | 5 | 4 | 3 | 5 | 4 |
| logs/dmesg | A | 3.9 | 4 | 5 | 3 | 3 | ? | ? |

Also known from FR-010 (overall only): network/nslookup C 2.6, network/arp 3.2, processes/delete 3.2, network/ping 3.3, services/list 3.3.

Reading the pattern (only `files/create`'s reasons are quoted by Glama; the rest is inferred and must be confirmed per tool):

- **Behavior** is the weakest dimension (2-3 on most tools): the descriptions do not say what needs `privileged`/a grant, what happens on error, whether parents are created, symlinks followed, recursion default, output truncated. For `files/create` Glama says exactly that: "With no annotations, the description carries full behavioral disclosure burden ... omits privileged/root writing, permission requirements, error behavior, or whether parent directories are created."
- **Parameters** sits at a flat 3 when the schema already documents every parameter and the description adds nothing: it wants defaults, precedence (e.g. line-based vs byte-based reads in `files/read`), formats and interactions.
- **Usage Guidelines** is low where a sibling exists and is not named (`files/create` vs `files/update`) and 5 where the description routes to named alternatives (`disks/list` -> `disks/free`, `disks/usage`).
- **Completeness** is low for tools with many parameters or unclear return shape (`files/find`: 9 parameters, one sentence, Completeness 1).
- **Annotations and output schemas are absent everywhere.** mcpd's `tools/list` carries no MCP `annotations` (`readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint`, `title`) and no `outputSchema` (verified: no such keys in `internal/rpc/tools.go` or `docker.go`). Glama's rubric says "no annotations are provided" repeatedly; annotations take the read-only/destructive/idempotent facts off the description, and an output shape spares the description from carrying it. This is the largest single lever and is mechanical.

### Work

1. **Per-tool analysis** (a table per tool: current description, known scores, what each weak dimension needs, and the true facts to state from the tool's code) - recorded in this ticket below "Analysis" before any rewriting.
2. **Annotations**: add MCP `annotations` to every tool schema in `tools/list` (registry-level, in `tools.go` and `docker.go`), set from what each tool really does; a Go test forbids a tool without them.
3. **Description rewrite** of every tool to the checklist in GUIDELINES.md §9 - all 46 tools, not just the 8 in FR-010 - keeping each tool's `README.md` and docs page in step.
4. **Guide for future tools**: GUIDELINES.md §9 and the pointer in CLAUDE.md step 3 of "Adding or changing a tool" (done with this ticket).
5. **Optional, later**: `outputSchema` for the tools that return JSON in `output_format: json`.

## Acceptance criteria

- [x] Guide written: GUIDELINES.md §9 (checklist per dimension, annotations rule, "every statement must be true", post-release check) and CLAUDE.md step 3 points to it.
- [ ] Analysis section in this ticket covers every tool (all 46), with the per-dimension needs and the facts to state.
- [ ] Every tool in `tools/list` has `annotations` set correctly; a Go test enforces it for the registry (existing and future tools).
- [ ] Every description rewritten to the checklist; each statement re-read against the tool's code (T2); READMEs and docs pages updated to match; `check_docs.sh` and `check_readmes.sh` pass.
- [ ] Released; Glama's next scoring shows every tool at grade A and no dimension at 2 or below - or the remaining gap per tool is recorded here.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Annotations everywhere | Go test over the registry (`HandleToolsList` output for a granted user): every tool has `annotations` with the four hints and a `title`; read-only tools are `readOnlyHint: true`, mutating ones are not | no tool without annotations | | |
| T2 | No false claims | each rewritten description re-read against the tool's code by a second pass | every statement true | | |
| T3 | Live | `linuxctl get mcp-api tools` and an MCP `tools/list` on a stand | new text and annotations present | | |
| T4 | Glama | tool page after the release (all 46, dimension by dimension); also read the 17 tools this ticket could not see | grade A everywhere, no dimension <= 2; remaining gaps recorded | | |
| T5 | Docs | `check_docs.sh`, `check_readmes.sh`, docs build | pass | | |

## Analysis

(to be filled - see Work step 1)

## Comments

- 2026-09-29 - created and in-progress at the owner's request. Fetched Glama's page: six dimensions, the 20 tool rows above, and the rubric's repeated "no annotations are provided" / no output schemas. Guide written the same day (GUIDELINES.md §9, CLAUDE.md step 3). FR-010 (8 lowest tools) is covered by this ticket; it is left in `new/` with a comment for the owner to close. Per-tool analysis handed to a background agent.
