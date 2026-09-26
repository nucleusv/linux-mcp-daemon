---
sidebar_position: 3
---

# AI Agent Configuration

The Linux MCP Daemon exposes a Linux host to AI agents over the **Model Context Protocol (MCP)**, with Server-Sent Events (SSE) as the transport. Any MCP client that speaks SSE can connect - Claude Code, Claude Desktop, Cursor, and others.

An agent needs two things:

- the SSE endpoint: `https://<host>:9091/sse` (TLS is on by default);
- a bearer token, sent as the `Authorization: Bearer <token>` header - one per MCP user, created with `linuxctl create mcpd user` (see [Installation](./installation)).

No daemon, no port, no token? A client can also start mcpd itself as a local process - on its own machine, over SSH (`ssh host mcpd stdio`) or inside a container - see [Stdio mode](./configuration/stdio-mode). It never runs anything as root.

## Trusting mcpd's certificate

By default mcpd serves a **self-signed** certificate it generated on first start (`/etc/mcpd/configs/tls/mcpd.crt`). A client has to trust it, or the TLS handshake fails:

- **Copy the certificate** to the machine the agent runs on and point the client at it. Clients built on Node.js (Claude Code, Claude Desktop, most `npx`-launched MCP bridges) read extra trusted certificates from `NODE_EXTRA_CA_CERTS`:
  ```bash
  scp root@my-host:/etc/mcpd/configs/tls/mcpd.crt ~/.config/mcpd/my-host.crt
  export NODE_EXTRA_CA_CERTS=~/.config/mcpd/my-host.crt
  ```
  The certificate covers the host's name, `localhost` and every address of the machine; for another name (a DNS alias), add it to `server.tls.hosts` in `daemon.yaml`, delete both files in `configs/tls/` and restart mcpd.
- **Or use a certificate from a real CA** (e.g. Let's Encrypt) that clients already trust: put its files at `server.tls.cert_file`/`key_file` in `daemon.yaml` (or point those at them) and restart mcpd. It then serves that one; nothing needs configuring on the clients.

Compare the certificate's fingerprint with the one mcpd logged at startup (`linuxctl describe mcpd tls` on the host) before trusting a copy.

## Claude Code

```bash
export NODE_EXTRA_CA_CERTS=~/.config/mcpd/my-host.crt    # when launching claude
claude mcp add --transport sse linux-my-host https://my-host:9091/sse \
  --header "Authorization: Bearer <token>"
```

## Claude Desktop and other `mcp_config.json`-style clients

```json
{
  "mcpServers": {
    "linux-my-host": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/client-sse",
        "--url",
        "https://my-host:9091/sse",
        "--header",
        "Authorization: Bearer <token>"
      ],
      "env": {
        "NODE_EXTRA_CA_CERTS": "/Users/me/.config/mcpd/my-host.crt"
      }
    }
  }
}
```

## Plain HTTP

Only on a trusted network, or behind a reverse proxy that terminates TLS: enable `server.http` in `daemon.yaml` (off by default) and use `http://<host>:9090/sse`. The bearer token then crosses the network in clear text with every request.

## What the agent may do

A token authenticates an MCP user, and every tool call runs as that user's own OS account - with no entry in `mcp-sudo.yaml` the agent can call every tool, but only with that account's permissions (it can't read `/root`, signal other users' processes, write system files). Running a tool **as root** (`privileged: true`) needs an explicit grant per tool, limited by paths, network destinations or sysctl keys - see [mcp-sudo.yaml](./configuration/mcp-sudo). Grant only what the agent's job needs.
