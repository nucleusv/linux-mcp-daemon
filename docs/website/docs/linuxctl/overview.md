---
sidebar_label: 'Overview'
sidebar_position: 1
---

# linuxctl CLI

The `linuxctl` command-line interface allows users and AI agents to seamlessly interact with the Linux Model Context Protocol (MCP) Daemon (`mcpd`) running in the background.

`linuxctl` speaks a small, uniform verb/group grammar (design rationale: `plan/linuxctl-redesign.md`) instead of a flat, one-off command per tool - every tool, resource, and resource template `mcpd` exposes is reachable through the same handful of words, discovered dynamically from the daemon's live schema on every invocation. Adding a new tool server-side makes it usable via `linuxctl` immediately, with no client code changes.

This section is split into a few focused pages so every way of using `linuxctl` is easy to find:

- **[Grammar & Commands](./grammar)** - the `<verb> <group> [target-keyword]` grammar itself, plus every standalone command (`ping`, `explain`, `resource`, `tool`).
- **[MCP Meta-Group](./mcp-meta-group)** - `get mcp tools/resources/prompts/info`, for introspecting the MCP protocol itself (what this daemon does and doesn't implement).
- **[Full Command Reference](./command-reference)** - every tool, resource, and template, grouped and with real captured output.
- **[Daemon User Administration](./mcpd-admin)** - `create`/`delete`/`update`/`list`/`describe mcpd user`, the local-only (no network) way to manage bearer tokens and permissions.

## Synopsis

```bash
linuxctl [OPTIONS] <verb> <group> [target-keyword] [args]
linuxctl [OPTIONS] describe <group> [target-keyword] <name>
linuxctl [OPTIONS] explain <group>
linuxctl [OPTIONS] get mcp <tools|resources|prompts|info>
linuxctl [OPTIONS] tool <group>/<command> [--flag val ...]
linuxctl [OPTIONS] resource <uri>
linuxctl [OPTIONS] <verb> mcpd user <username>
```

## Options

- `-server URL`
  The URL of the `mcpd` server to connect to. Defaults to `http://localhost:9091`.

- `-token TOKEN`
  Bearer token for authentication. If not provided via this flag, the client looks for the `MCP_TOKEN` environment variable. Not needed for the local-only `mcpd` admin group.

- `-config-path DIR`
  Local path to the daemon's `configs/` directory. Only used by the local-only `mcpd` admin group - defaults to `./configs`.

## Environment Variables

- `MCP_TOKEN`
  The bearer token used for authenticating with the daemon. This is the recommended way to authenticate so you don't leak tokens in shell histories.

## Quick example

```bash
export MCP_TOKEN="your_token_here"
linuxctl ping
linuxctl get files /etc/hosts
linuxctl get disks free /
```

See [Grammar & Commands](./grammar) for the full rules, or jump straight to the [Full Command Reference](./command-reference) for real, copy-pasteable examples across every tool.
