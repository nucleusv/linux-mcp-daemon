---
sidebar_position: 1
---

# Master Daemon Configuration

The core settings for the Linux MCP Daemon are defined in `configs/daemon.yaml`.

This file controls the global web server configuration, connection timeouts, rate limiting, and user authentication tokens.

## Example `daemon.yaml`

```yaml
server:
  port: 8080
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/mcpd.crt"
    key_file: "/etc/ssl/private/mcpd.key"

worker:
  timeout_seconds: 30

rate_limits:
  default_rps: 5
  default_burst: 10

tools:
  get_disk_usage:
    timeout_seconds: 120

users:
  - username: "alice"
    token: "secret123"
  - username: "bob"
    token: "secret456"
```

## Settings breakdown

- **`server.port`**: The HTTP port that the daemon listens on for `/sse`, `/message`, and `/docs/` traffic.
- **`server.tls`**: If `enabled` is true, the daemon natively hosts an HTTPS server using the provided `cert_file` and `key_file`. If false, it serves plain HTTP (which you should place behind a reverse proxy like NGINX).
- **`worker.timeout_seconds`**: The global maximum time an Ephemeral Worker is allowed to run before the Master daemon sends a `SIGKILL`. This prevents runaway processes.
- **`rate_limits`**: Global rate limits applied to every authenticated user to prevent an AI from spamming the server and saturating your I/O.
- **`tools.<name>.timeout_seconds`**: Tool-specific overrides. Heavy tools like `get_disk_usage` can be granted longer execution windows than lightweight tools.
- **`users`**: A static list of users and their Bearer tokens. The `username` is critical because it binds to the `mcp-sudo.yaml` file to determine privileges.
