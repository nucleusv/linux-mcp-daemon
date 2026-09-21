---
sidebar_position: 1
---

# Ephemeral Workers

The most critical security feature of the Linux MCP Daemon is its **Ephemeral Worker** architecture.

When an AI requests a tool execution, the Master daemon **never** executes that logic itself. The Master daemon runs as `root` so it can orchestrate network connections and spawn processes, but running AI-generated paths directly in the root process could lead to devastating Path Traversal vulnerabilities.

## How it works

1. The Master daemon receives an MCP `tools/call` JSON-RPC message.
2. It authenticates the user via their Bearer token.
3. Instead of running the tool, the Master uses `os/exec` to spawn a new instance of itself (`./mcpd worker <tool> <json>`).
4. Critically, it attaches a `syscall.Credential` to the `Cmd.SysProcAttr` object containing the exact UID and GID of the authenticated user.
5. The operating system drops all privileges and starts the worker process securely.
6. The worker executes the tool, prints the result to `stdout`, and immediately dies.
7. The Master captures the stdout via an `io.Pipe` and sends it back to the AI.

This guarantees that the AI can only ever access files that the human user actually has permission to access on the host operating system!
