---
sidebar_position: 3
---

# AI Agent Configuration

The primary purpose of the Linux MCP Daemon is to securely expose the Linux operating system to AI agents. It does this via the open standard **Model Context Protocol (MCP)** using Server-Sent Events (SSE).

You can connect almost any modern AI agent or IDE (such as Antigravity IDE, Cursor, or Claude Desktop) directly to this daemon.

## Connecting via SSE (Server-Sent Events)

To connect an AI agent to the daemon, you must use an MCP SSE client that connects to the `http://localhost:9091/sse` endpoint and provides the necessary Bearer token.

### Example: Antigravity IDE / Generic Config
If your agent supports standard `mcp_config.json` structures, you can use the official `@modelcontextprotocol/client-sse` package via `npx`:

```json
{
  "mcpServers": {
    "linux-mcp-daemon": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/client-sse",
        "--url",
        "http://localhost:9091/sse",
        "--header",
        "Authorization: Bearer my-test-token-123"
      ]
    }
  }
}
```

### Example: Claude Desktop Configuration

For Claude Desktop, update your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "linux-mcp-daemon": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/client-sse",
        "--url",
        "http://localhost:9091/sse",
        "--header",
        "Authorization: Bearer my-test-token-123"
      ]
    }
  }
}
```

## Security & Authorization

By default, connecting to the daemon grants the AI agent limited, read-only introspection capabilities. 

If you want your agent to execute privileged operations (e.g. `disks/free` or `system/manage`), you must explicitly whitelist those commands for the specific token/user in the `configs/mcp-sudo.yaml` file.

See the [Configuration/MCP Sudo](./configuration/mcp-sudo.md) documentation for details on privilege escalation.
