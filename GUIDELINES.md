# Linux MCP Daemon - Agent Guidelines & Code Agreements

This document outlines the core architectural principles, naming conventions, and security guidelines for the `linux-mcp-daemon` project. AI Agents and human contributors MUST adhere to these guidelines when modifying the codebase.

## 1. Naming Conventions & Structure
*   **Tool Naming**: All tools MUST use the path-based naming convention: `<group>/<command>`. 
    *   *Correct*: `files/list`, `disks/free`, `network/ping`, `system/os-release`
    *   *Incorrect*: `list_files`, `get_free`, `network/get_ping`, `read_os_release`
*   **Directory Structure**: Tool implementations live under `internal/tools/<group>/<command>/`. Intermediate verb directories (like `/get/`, `/read/`, or `/delete/`) MUST NOT be used.
    *   *Correct*: `internal/tools/disks/usage/usage.go`
    *   *Incorrect*: `internal/tools/disks/get/usage/usage.go`
*   **Package Naming**: Go package names should avoid hyphens and reserved keywords. For example, `processes/delete` uses package `deleteprocess`, and `cpu/load-average` uses package `loadaverage`.

## 2. Architecture: Ephemeral Workers
*   **Strict Isolation**: The main daemon loop (`mcpd`) MUST NOT directly read system files (e.g., `/proc`, `/sys`) or execute external binaries.
*   **Worker Spawn**: All hardware reads, system queries, and tool executions MUST be routed through isolated ephemeral workers via `worker.SpawnWorker(toolName, args)`. 
*   **Worker Mode**: The worker executes the tool (via `mcpd worker <tool_name> <args>`), returns JSON to `stdout`, and terminates immediately.

## 3. Modular Codebase
*   **No Monoliths**: Avoid adding extensive logic to `cmd/mcpd/main.go`. The daemon is split logically (e.g., `types.go`, `http.go`, `rpc.go`).
*   **Adding New Tools**:
    1. Create the logic under `internal/tools/<group>/<command>/<command>.go`.
    2. Add the tool switch case in `cmd/mcpd/main.go` (in the `mcpd worker` block).
    3. Register the tool in the JSON-RPC schema (`getToolsList`) inside `cmd/mcpd/rpc.go`. Make sure to include `"tools_group": "<group>"`.
    4. Ensure the tool is handled in the main execution router (`tools/call` handler in `cmd/mcpd/rpc.go`).

## 4. CLI (`linuxctl`)
*   **Direct Mapping**: The CLI maps arguments directly to the backend. Running `linuxctl <group> <command>` evaluates exactly to the tool name `<group>/<command>`. 
*   **No Smart Routers**: Do not use suffix matching or fuzzy logic to map CLI commands to MCP tools.

## 5. Security & Sudo (`mcp-sudo.yaml`)
*   **Privilege Check**: The daemon MUST intercept and authorize all `tools/call` and `resources/read` requests against `configs/mcp-sudo.yaml`.
*   **Paths**: Always configure `<group>/<command>` explicitly in `mcp-sudo.yaml`. For destructive or sensitive tools (like `files/list` or `processes/delete`), restrict by `Paths` or completely disable them for unprivileged users.

## 6. Caching
*   Static or slow-changing hardware information (like DMI data, OS Release, or PCI listings) should utilize the internal cache system to minimize the overhead of frequently spawning ephemeral workers.
*   Dynamic data (like CPU usage, processes, active connections) MUST bypass the cache and query live data.

## 7. Documentation & Code Comments
*   **Arg Structures**: Every tool's argument struct (e.g. `type Args struct { ... }`) MUST be fully commented. Explain what each field does, what defaults apply if omitted, and document any required input validations.
*   **JSON Struct Tags**: When defining JSON struct tags (e.g. `` `json:"output_format,omitempty"` ``), ensure you do not use escaped backslashes `\"` inside the raw string literal backticks, as this breaks Go's reflection parser and causes the CLI arguments to be ignored.
*   **GoDoc Standard**: Maintain rigorous GoDoc comments on all exported functions, types, and structs across the codebase to ensure automatic documentation generators provide meaningful output.
*   **Developer Documentation**: The website documentation (`docs/website/`) is tightly coupled to the codebase. When introducing new tools, behaviors, or privileges, you MUST correspondingly create or update the relevant Markdown documentation in `docs/website/docs/` based on your commits.
*   **README Requirements**: If you modify tool code, add a new tool, or rename an existing tool, you MUST update the associated `README.md` (both the tool's individual README and the main project README if applicable) to reflect the correct command names, configurations, and arguments.

## 8. Testing Strategy
*   **Go Unit Tests**: Standard library `testing` is used for internal business logic and modular components (e.g., `config`, `auth`). Keep tests adjacent to the code they verify (`config_test.go`).
*   **Integration Tests**: Bash scripts under `tests/` (like `test_mcp.sh` and `test_linuxctl.sh`) perform end-to-end testing against a compiled daemon and CLI instance. Ensure these are updated whenever modifying tool schemas or command line routing.

## 9. Tool descriptions and annotations (Glama TDQS)

Glama scores every tool on six dimensions, 1-5 each, from the `description` and the schema that `tools/list` returns (page: https://glama.ai/mcp/servers/nucleusv/linux-mcp-daemon). Agents choose tools the same way the scorer reads them, so a description is written for a model that sees only that text. When you add or change a tool, its `description` in `internal/rpc/tools.go` must pass this checklist:

| Dimension | What the description must do |
|---|---|
| **Purpose** | First sentence: verb + resource, and how it differs from its siblings, by name (`files/create` vs `files/update`, `disks/usage` vs `disks/free`). |
| **Usage guidelines** | When to use it, when not to, and the alternative: "For X use `group/tool`". Scores of 5 come from descriptions that route to named alternatives. |
| **Behavior** | Read-only or mutating; what needs `privileged: true` or a grant, and what is refused without it; the edge behavior an agent will hit - creates parent directories? follows symlinks? recursive by default? truncates output? timeout? idempotent? |
| **Parameters** | Only what the schema does not already say: defaults, units, formats, precedence between parameters, mutually exclusive ones, one example of non-obvious syntax (a glob, a range). Repeating the schema scores a flat 3. |
| **Completeness** | What comes back: the shape of the output (text vs `output_format: json`), limits and truncation, what an empty result means. mcpd has no `outputSchema`, so the description carries this. |
| **Conciseness** | Front-loaded, no filler or marketing, every sentence carries a fact. Typical length 60-150 words; long only when the tool has many interacting parameters. |

Rules that go with it:

- **Every statement must be true** against the tool's code (re-read the description against the implementation before committing). A wrong claim costs an agent a failed call.
- **Annotations:** every tool carries MCP `annotations` in `tools/list` - `title`, `readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint` - set from what the tool really does (a read-only tool is `readOnlyHint: true`; `files/create`, `processes/delete`, `docker/manage`, `docker/prune` are destructive). Glama's rubric says "no annotations are provided" for tools without them, which forces the description to carry the whole behavioral burden. (Introduced by FR-024; until it is done, describe read/write behavior in the text.)
- **Check the score after a release:** open the Glama tool list; a tool under 4.0 on any dimension gets a follow-up. Record the result in the ticket.
- The docs page and README of the tool repeat the description: change all three together (`check_docs.sh`, `check_readmes.sh`).

