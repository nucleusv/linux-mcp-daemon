---
sidebar_label: 'MCP Meta-Group'
sidebar_position: 3
---

# The `mcp` meta-group

`get mcp <tools|resources|prompts|info>` is a reserved meta-group for the MCP protocol's own top-level catalog concepts - not a real `tools_group`, and distinct from `tool <name>`/`resource <uri>` (which invoke one specific item). It exists to answer "what can we get from this daemon over the MCP standard itself," straight from the protocol:

```bash
$ linuxctl get mcp tools
Available tools:
  files/list                - Lists contents of a directory. ...
  files/read                - Precision reading of file contents ...
  ...

$ linuxctl get mcp resources
Available static resources:
  os://uname           - Native system uname information ...
  ...

$ linuxctl get mcp prompts
This mcpd daemon does not implement the MCP prompts capability (Method not found).
Its initialize response only declares "tools" and "resources" - see `linuxctl get mcp info`.

$ linuxctl get mcp info
{
  "capabilities": {
    "resources": {},
    "tools": {}
  },
  "protocolVersion": "2024-11-05",
  "serverInfo": {
    "name": "linux-mcp-daemon",
    "version": "1.0.0"
  }
}
```

`get mcp info` calls the real MCP `initialize` method and prints the raw response - the server's own self-declared protocol version and capabilities, not a client-side summary of them. `get mcp prompts` calls the real `prompts/list` method too, rather than silently reporting an empty list: since `mcpd` genuinely doesn't implement the MCP `prompts` capability (only `tools` and `resources` are declared in its `initialize` response - see `internal/rpc/rpc.go`), the honest answer is to say so plainly.

## What `mcpd` does and doesn't implement

The full MCP spec (2024-11-05, the version this daemon declares) has a few other server-side capabilities beyond tools/resources/prompts, none of which `mcpd` implements:

| Capability | Implemented? | Notes |
|---|---|---|
| `tools` | ✅ | `tools/list`, `tools/call` |
| `resources` | ✅ (partial) | `resources/list`, `resources/read`, `resources/templates/list` - but not `subscribe`/`unsubscribe` (see below) |
| `prompts` | ❌ | Not declared in `initialize`; `get mcp prompts` reports this plainly |
| `resources.subscribe`/`unsubscribe` | ❌ | Push/watch mechanism - the server notifies the client when a specific resource changes, instead of the client re-polling `resources/read`. Deliberately not planned: it would require the daemon to hold a persistent background watcher (e.g. inotify) per subscription, which doesn't fit the current architecture where every tool/resource call spawns a fresh, short-lived worker process and exits (`internal/worker/spawner.go`) - there's no long-lived process to hold the watch. |
| `logging` | ❌ | The server pushing log messages to the client via `notifications/message` |
| `completion` | ❌ | `completion/complete`, argument autocompletion for prompt/resource-template parameters |

`roots` and `sampling` are *client*-side capabilities in the spec - the client declaring filesystem roots or offering its own LLM back to the server - so they're not something `mcpd` would ever expose as a server, and don't belong in this table at all.
