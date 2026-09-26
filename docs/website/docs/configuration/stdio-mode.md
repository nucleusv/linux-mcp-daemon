---
sidebar_position: 5
---

# Stdio mode (`mcpd stdio`)

`mcpd stdio` serves MCP over stdin/stdout instead of the network: the MCP client starts it as a local process, the way most MCP servers are run. The tools, resources and their output are exactly the same as over HTTPS - only the transport differs.

```bash
mcpd stdio [--user NAME] [--config-dir DIR]
```

## When it is useful

| Situation | How |
|---|---|
| **The agent's own machine** - a laptop or dev box. No daemon, no port, no TLS, no tokens. | client runs `mcpd stdio` |
| **A remote server with no open port.** Authentication is your existing SSH key; nothing listens on the network, and the agent still gets typed tools instead of a shell. | client runs `ssh admin@web1 mcpd stdio` |
| **Inside a container or a pod** - look at it from the inside, as its own user. | `docker exec -i app mcpd stdio --user app`, `kubectl exec -i pod -- mcpd stdio --user app` |
| **Trying mcpd out** in a minute, before setting up the daemon, users and TLS. | `mcpd stdio` |
| **MCP Inspector, CI, catalogs' build checks** - anything that starts a server as a process. | `npx @modelcontextprotocol/inspector mcpd stdio` |
| **Locked-down hosts** where opening a port is not allowed. | over SSH, as above |

When **not** to use it: several people or agents sharing one host, a separate identity per agent, root for specific tools, one audit log for everyone. That is what the [network daemon](../installation) is for.

:::caution Over SSH, it limits the agent - not the key
`ssh host mcpd stdio` gives the *agent* only mcpd's tools. The SSH key it uses still opens a full shell for anyone who has it. Keep that key for this purpose only, or restrict it on the server with `command="mcpd stdio"` in `authorized_keys`.
:::

## Security model

Over stdio there is no token and no MCP user, so the rules are simpler and stricter than the daemon's:

- **The caller is an OS account.** Started by an ordinary user, every tool call runs as that user - whatever the kernel lets them do, the agent can do, and nothing more.
- **Never root.** `privileged: true` is always refused, and `mcp-sudo.yaml` is not read. A root grant over stdio would belong to whoever controls the client's config, which is exactly the [stdio risk](https://habr.com/ru/news/1025116/) MCP clients have.
- **Started as root** (`docker exec`, a catalog's container), mcpd refuses to serve unless `--user NAME` names an unprivileged account; each call then runs as NAME, like the daemon's workers. `--user root` is refused.
- **stdout carries only MCP messages**; mcpd's own log goes to stderr, in the same format as the daemon's (`session=stdio`).

A refused root call looks like this:

```text
$ echo '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"files/read","arguments":{"path":"/etc/shadow","privileged":true}}}' | mcpd stdio --user agent
{"jsonrpc":"2.0","id":4,"result":{"content":[{"type":"text","text":"privileged: true is not available when mcpd serves over stdio - it never runs anything as root. For root on specific tools, use the mcpd network daemon with a grant in mcp-sudo.yaml"}],"isError":true}}
```

And starting as root without `--user`:

```text
$ sudo mcpd stdio
ERROR cannot serve over stdio err="started as root: pass --user NAME (an unprivileged account) - mcpd stdio never runs tools as root"
```

## Connecting a client

**Claude Code**

```bash
claude mcp add linux -- mcpd stdio                          # this machine
claude mcp add web1 -- ssh admin@web1 mcpd stdio            # a remote server, over SSH
claude mcp add app -- docker exec -i app mcpd stdio --user app   # inside a container
```

**Claude Desktop and other `mcp_config.json`-style clients**

```json
{
  "mcpServers": {
    "web1": {
      "command": "ssh",
      "args": ["admin@web1", "mcpd", "stdio"]
    }
  }
}
```

**MCP Inspector** - a quick check that everything answers. Here mcpd runs in a container, reached with `docker exec -i` through a two-line wrapper (`exec docker exec -i stdio-t mcpd stdio --user agent`):

```text
$ npx @modelcontextprotocol/inspector --cli ./mcpd-stdio-docker.sh --method tools/list
37 tools: files/list, files/read, files/create, files/update, ...
$ npx @modelcontextprotocol/inspector --cli ./mcpd-stdio-docker.sh --method resources/list
11 resources
$ npx @modelcontextprotocol/inspector --cli ./mcpd-stdio-docker.sh --method tools/call --tool-name memory/usage
              total         used         free       shared   buff/cache    available
...
```

(Output trimmed; the Inspector prints the full JSON.)

## Configuration

Nothing is required. If `--config-dir` (or `MCPD_CONFIG_DIR`, or `configs/` in the working directory) holds a `daemon.yaml`, stdio mode uses only its worker timeout, per-tool timeouts (`tools:`) and `logging:`. Listeners, users, TLS and `mcp-sudo.yaml` belong to the network daemon and are ignored here - so are the `network:` limits in `mcp-sudo.yaml`: `network/curl` over stdio can reach whatever the calling user can.

## See also

- [AI Agent Configuration](../ai-agent-configuration) - the network daemon with Claude Code and Claude Desktop
- [Permissions and Risks](./permissions-and-risks) - what root grants mean, for the daemon
