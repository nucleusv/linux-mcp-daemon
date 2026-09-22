---
sidebar_position: 2
---

# Sudo Privileges

Unlike traditional API servers, the Linux MCP Daemon implements a secure Privilege Escalation mechanism similar to Unix `sudoers`. This is controlled entirely by `configs/mcp-sudo.yaml`.

## How it works

When an AI sends a tool execution request (e.g., `files/list`), it can optionally attach `"privileged": true` to the JSON-RPC arguments.

If the AI attempts to run a tool as root, the Master daemon intercepts the request and checks the `mcp-sudo.yaml` file to see if the authenticated user (bound by their session token) is authorized to elevate privileges for that specific tool.

If authorized, the Ephemeral Worker spawns natively as UID 0. If denied, the request is blocked before a worker is ever spawned.

## Example `mcp-sudo.yaml`

```yaml
users:
  alice:
    privileged:
      tools:
        disks/free:
          allowed: true
        files/list:
          allowed: true
          paths:
            - "/var/log"
            - "/root"
      resources:
        devices://usb:
          - "*"
  bob:
    privileged:
      tools:
        files/list:
          allowed: true
          paths:
            - "/var/www"
        disks/usage:
          allowed: true
```

In this example, if the AI is authenticated as `alice`, it can list protected root directories, but it cannot run expensive `disks/usage` tree traversals as root!

## Full reference example

The daemon's own `configs/mcp-sudo.yaml` includes a `privileged` user with every tool and every resource this daemon currently exposes granted. It isn't meant to represent a real least-privilege user (see `admin`/`testuser`/`unpriviliged` in that same file for that) - it exists purely as living documentation of the complete authorization surface, and is kept in sync with `internal/rpc/tools.go`/`resources.go` whenever a tool or resource is added, renamed, or removed.

## Host filesystem access

When this daemon runs containerized (see [Master Daemon Configuration](./daemon.md)'s `worker.containerized` setting), `privileged: true` means more than root-in-container: every privileged worker call automatically also joins the real host's mount namespace and chroots into it, so tools like `system/packages` or `services/manage` see the actual host filesystem and the actual host's systemd, not the daemon's own container image. This is entirely a daemon-startup setting - there's nothing to configure per-tool here, and no second flag alongside `privileged: true` to authorize. If `mcpd` runs directly on the host instead (no container boundary), the same `privileged: true` grant just runs as root normally, since there's nothing else to cross into.
