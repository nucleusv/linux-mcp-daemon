---
sidebar_position: 1
---

# Master Daemon Configuration

The core settings for the Linux MCP Daemon are defined in `configs/daemon.yaml`.

This file controls the global web server configuration, connection timeouts, rate limiting, and user authentication tokens.

## Example `daemon.yaml`

```yaml
server:
  port: 9091
  tls:
    enabled: true
    port: 9443
    cert_file: "/etc/ssl/certs/mcpd.crt"
    key_file: "/etc/ssl/private/mcpd.key"

worker:
  timeout_seconds: 30
  containerized: true

rate_limits:
  default_rps: 5
  default_burst: 10

tools:
  disks/usage:
    timeout_seconds: 120

users:
  - username: "alice"
    token: "secret123"
  - username: "bob"
    token: "secret456"
```

## Settings breakdown

- **`server.port`**: The HTTP port that the daemon listens on for `/sse`, `/message`, and `/docs/` traffic.
- **`server.tls`**: If `enabled` is true, the daemon natively hosts a concurrent HTTPS server on `tls.port` (default `9443`) using the provided `cert_file` and `key_file`. The plain HTTP server will continue to run simultaneously on `server.port`!
- **`worker.timeout_seconds`**: The global maximum time an Ephemeral Worker is allowed to run before the Master daemon sends a `SIGKILL`. This prevents runaway processes.
- **`worker.containerized`**: Set to `true` when `mcpd` itself runs inside a container (e.g. Kubernetes, Docker) with its own private root filesystem, as this daemon's own deployment does (see `k8s/deployment.yaml`). When true, every `privileged: true` tool call also joins the real host's mount namespace before running, so tools like `system/packages` or `services/manage` administer the actual host rather than the daemon's own container image. Set `false` when `mcpd` runs directly on the host with no container boundary to cross - `mcpd` also self-checks this at startup and logs a warning if the configured value doesn't match what it detects about its own environment.
- **`rate_limits`**: Global rate limits applied to every authenticated user to prevent an AI from spamming the server and saturating your I/O.
- **`tools.<name>.timeout_seconds`**: Tool-specific overrides, keyed by the tool's full `<group>/<command>` name. Heavy tools like `disks/usage` can be granted longer execution windows than lightweight tools.
- **`users`**: A static list of users and their Bearer tokens. The `username` is critical because it binds to the `mcp-sudo.yaml` file to determine privileges.
