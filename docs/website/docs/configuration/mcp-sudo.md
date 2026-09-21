---
sidebar_position: 2
---

# Sudo Privileges

Unlike traditional API servers, the Linux MCP Daemon implements a secure Privilege Escalation mechanism similar to Unix `sudoers`. This is controlled entirely by `configs/mcp-sudo.yaml`.

## How it works

When an AI sends a tool execution request (e.g., `list_directory`), it can optionally attach `"privileged": true` to the JSON-RPC arguments.

If the AI attempts to run a tool as root, the Master daemon intercepts the request and checks the `mcp-sudo.yaml` file to see if the authenticated user (bound by their session token) is authorized to elevate privileges for that specific tool.

If authorized, the Ephemeral Worker spawns natively as UID 0. If denied, the request is blocked before a worker is ever spawned.

## Example `mcp-sudo.yaml`

```yaml
users:
  alice:
    privileged:
      get_disk_free:
        allowed: true
      list_directory:
        allowed: true
        paths:
          - "/var/log"
          - "/root"
  bob:
    privileged:
      list_directory:
        allowed: true
        paths:
          - "/var/www"
      get_disk_usage:
        allowed: true
```

In this example, if the AI is authenticated as `alice`, it can list protected root directories, but it cannot run expensive `get_disk_usage` tree traversals as root!
