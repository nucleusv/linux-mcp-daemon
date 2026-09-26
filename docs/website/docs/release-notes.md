---
sidebar_position: 99
sidebar_label: 'Release Notes'
---

# Release Notes

What changed in each release, and what to do when upgrading. Every release, with its binaries, packages and the full commit list, is on [GitHub Releases](https://github.com/nucleusv/linux-mcp-daemon/releases).

## 0.3.5

Upgrading from 0.3.4 needs no changes: configs, tools and their output are the same.

### New: stdio mode

`mcpd stdio` serves MCP over stdin/stdout, for clients that start a server as a local process - no daemon, no port, no token, no TLS:

```bash
claude mcp add linux -- mcpd stdio                                  # this machine
claude mcp add web1 -- ssh admin@web1 mcpd stdio                    # a remote server over SSH, no open port
claude mcp add app -- docker exec -i app mcpd stdio --user app      # inside a container
```

Tool calls run as the OS user who started mcpd and **never as root**: `privileged: true` is refused and `mcp-sudo.yaml` is not read. Started as root (`docker exec`, a catalog's build), mcpd requires `--user NAME` and runs each call as that account. When to use it and when the network daemon is the better fit: [Stdio mode](./configuration/stdio-mode).

### Fixed

- **Notifications are no longer answered.** A message without an `id` - `notifications/cancelled`, for example - used to get a `Method not found` reply with `id: null`. JSON-RPC forbids replying to a notification; over stdio a client would read it as an unexpected message.

### Elsewhere

- mcpd is listed in the official [MCP Registry](https://registry.modelcontextprotocol.io/v0/servers?search=io.github.nucleusv/linux-mcp-daemon) as `io.github.nucleusv/linux-mcp-daemon`, and on [Glama](https://glama.ai/mcp/servers/nucleusv/linux-mcp-daemon).
