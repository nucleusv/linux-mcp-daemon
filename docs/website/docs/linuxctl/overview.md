---
sidebar_label: 'Overview'
sidebar_position: 1
---

# linuxctl CLI

The `linuxctl` command-line interface allows users and AI agents to seamlessly interact with the Linux Model Context Protocol (MCP) Daemon (`mcpd`) running in the background.

`linuxctl` speaks a small, uniform verb/group grammar (design rationale: `plan/linuxctl-redesign.md`) instead of a flat, one-off command per tool - every tool, resource, and resource template `mcpd` exposes is reachable through the same handful of words, discovered dynamically from the daemon's live schema on every invocation. Adding a new tool server-side makes it usable via `linuxctl` immediately, with no client code changes.

This section is split into a few focused pages so every way of using `linuxctl` is easy to find:

- **[Grammar & Commands](./grammar)** - the `<verb> <group> [target-keyword]` grammar itself, plus every standalone command (`ping`, `explain`, `resource`, `tool`).
- **[MCP Meta-Group](./mcp-meta-group)** - `get mcp-api tools/resources/prompts/info`, for introspecting the MCP protocol itself (what this daemon does and doesn't implement).
- **[Full Command Reference](./command-reference)** - every tool, resource, and template, grouped and with real captured output.
- **[Daemon User Administration](./mcpd-admin)** - `create`/`delete`/`update`/`list`/`describe mcpd user`, the local-only (no network) way to manage bearer tokens and permissions.

## Synopsis

```bash
linuxctl [OPTIONS] <verb> <group> [target-keyword] [args]
linuxctl [OPTIONS] describe <group> [target-keyword] <name>
linuxctl [OPTIONS] explain <group>
linuxctl [OPTIONS] get mcp-api <tools|resources|prompts|info>
linuxctl [OPTIONS] tool <group>/<command> [--flag val ...]
linuxctl [OPTIONS] resource <uri>
linuxctl [OPTIONS] <verb> mcpd user <username>
linuxctl completion <bash|zsh>
```

## Options

- `-server URL`
  The URL of the `mcpd` server to connect to. Defaults to the `MCP_SERVER` environment variable, or `https://localhost:9091` if that's unset.

- `-token TOKEN`
  Bearer token for authentication. If not provided via this flag, the client looks for the `MCP_TOKEN` environment variable. Not needed for the local-only `mcpd` admin group.

- `-config-path DIR`
  Local path to the daemon's `configs/` directory. Only used by the local-only `mcpd` admin group - defaults to `./configs`.

- `-silent`, `-s`
  Print no errors or warnings, like `curl -s` - for scripts. Output goes to stdout and errors to stderr either way; `-s` only discards stderr. The exit status still tells a failure: 0 on success, 1 on any error, a tool's own failure included:
  ```bash
  if ! linuxctl -s get files read --path /etc/app.conf > app.conf; then
      echo "could not read the config" >&2
  fi
  ```

- `-tls-insecure`, `-k`
  Skip TLS certificate verification, like `curl -k` (or `MCP_TLS_INSECURE=1`). Testing only - see below.

## Environment Variables

- `MCP_TOKEN`
  The bearer token used for authenticating with the daemon. This is the recommended way to authenticate so you don't leak tokens in shell histories.

- `MCP_SERVER`
  Default for `-server`. Prefer this over a shell alias like `alias linuxctl="linuxctl -server ..."` - shell completion can't see through aliases.

- `MCP_TLS_FINGERPRINT` (or `-tls-fingerprint`)
  Trust the server whose certificate has this SHA-256 fingerprint - the simplest way to trust mcpd's self-signed certificate from another machine. mcpd logs it at startup; `linuxctl describe mcpd tls` shows it on the daemon's host.

- `MCP_CA_CERT` (or `-ca-cert`)
  Trust this certificate file (PEM) - e.g. a copy of the host's `/etc/mcpd/configs/tls/mcpd.crt`. Without either, `linuxctl` trusts the system's CAs plus, on the daemon's own host, its certificate when readable.

- `MCP_TLS_INSECURE=1` (or `-tls-insecure`, short `-k` as in curl)
  Skip certificate verification. For testing only: any certificate is accepted, a forged one too, and the token is sent to it - pin the fingerprint instead.

## Quick example

```bash
export MCP_SERVER="https://my-host:9091"
export MCP_TLS_FINGERPRINT="sha256:..."   # linuxctl describe mcpd tls, on my-host
export MCP_TOKEN="your_token_here"
linuxctl ping
linuxctl get files /etc/hosts
linuxctl get disks free /
```

See [Grammar & Commands](./grammar) for the full rules, or jump straight to the [Full Command Reference](./command-reference) for real, copy-pasteable examples across every tool.

## Shell completion

`linuxctl completion bash|zsh` enables `Tab` completion of verbs, groups, keywords, flags and resource URIs, driven by the daemon's live registry. See [Autocompletion (bash/zsh)](./autocompletion) for setup.

## Table output width

`-o table` fits the terminal - long trailing columns such as a process's command line are cut with `…`, like `ps`. This still applies when piped (`| more`, `| less`), using the controlling terminal's width. Use `-o wide` for untruncated output, as with `kubectl`.
