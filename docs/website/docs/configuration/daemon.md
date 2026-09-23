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
    token_salt: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"
    token_hash: "9f8e7d6c5b4a...  # sha256(salt + plaintext token), hex"
    created_at: "2026-09-23T00:28:15Z"
  - username: "bob"
    token: "secret456"   # legacy plaintext form - still supported, but new/rotated users use token_hash instead
```

## Settings breakdown

- **`server.port`**: The HTTP port that the daemon listens on for `/sse`, `/message`, and `/docs/` traffic.
- **`server.tls`**: If `enabled` is true, the daemon natively hosts a concurrent HTTPS server on `tls.port` (default `9443`) using the provided `cert_file` and `key_file`. The plain HTTP server will continue to run simultaneously on `server.port`!
- **`worker.timeout_seconds`**: The global maximum time an Ephemeral Worker is allowed to run before the Master daemon sends a `SIGKILL`. This prevents runaway processes.
- **`worker.containerized`**: Set to `true` when `mcpd` itself runs inside a container (e.g. Kubernetes, Docker) with its own private root filesystem, as this daemon's own deployment does (see `k8s/deployment.yaml`). When true, every `privileged: true` tool call also joins the real host's mount namespace before running, so tools like `system/packages` or `services/manage` administer the actual host rather than the daemon's own container image. Set `false` when `mcpd` runs directly on the host with no container boundary to cross - `mcpd` also self-checks this at startup and logs a warning if the configured value doesn't match what it detects about its own environment.
- **`rate_limits`**: Global rate limits applied to every authenticated user to prevent an AI from spamming the server and saturating your I/O.
- **`tools.<name>.timeout_seconds`**: Tool-specific overrides, keyed by the tool's full `<group>/<command>` name. Heavy tools like `disks/usage` can be granted longer execution windows than lightweight tools.
- **`users`**: A list of users and their Bearer token credentials. The `username` is critical because it binds to `mcp-sudo.yaml` for privilege grants, **and** must match a real OS account in the container image (`SpawnWorker` does `user.Lookup()` against the OS passwd database to resolve a UID for every tool call - see `Dockerfile`'s `useradd` lines).

### Token storage: salted hash vs. legacy plaintext

New or rotated users store `token_salt` + `token_hash` (`sha256(salt + token)`, hex-encoded) instead of a plaintext `token`. The daemon supports both forms simultaneously (`authenticateRequest` in `cmd/mcpd/http.go` checks `token_hash` first, falls back to plaintext `token` if present) so migrating existing users doesn't require a hard cutover. **Don't hand-write a `token_hash` entry** - always go through `linuxctl` (see below), since the salt must be freshly random per user and the hash must exactly match `sha256(salt + token)` in hex or authentication will simply always fail for that account.

## Managing users with `linuxctl` (local-only, no daemon round-trip)

`linuxctl`'s `mcpd` group reads and writes `configs/daemon.yaml` + `configs/mcp-sudo.yaml` directly on disk - it never makes a network call to the running daemon, and needs no `-token`/`-server` flag at all. This is deliberate: user/token administration is a privilege-escalation-relevant surface, so it's kept off the network `tools/call` path entirely, reachable only to whoever has local filesystem access to these config files.

```bash
# Defaults to ./configs - override with --config-path if running from elsewhere
linuxctl create   mcpd user alice                    # generates a random token, prints it once
linuxctl create   mcpd user alice --set-token "abc"  # sets a specific token instead (used for migrating existing accounts)
linuxctl update   mcpd user alice                    # rotates to a new random token, prints it once
linuxctl delete   mcpd user alice                    # removes from BOTH files atomically
linuxctl list     mcpd users                         # username, created_at, and hash-vs-plaintext status - never the token itself
linuxctl describe mcpd user alice                    # that user's full mcp-sudo.yaml grant block
```

**Why `create`/`delete` touch both files atomically**: doing this by hand (editing `daemon.yaml` and `mcp-sudo.yaml` as two separate manual edits) is exactly how a stale privilege grant survives a user being "deleted" and later recreated under the same name - if only one file gets edited, the other file's old grants silently persist and apply to whoever gets that username next. `linuxctl create mcpd user` always writes a **fresh, empty** `mcp-sudo.yaml` block, overwriting any stale leftover entry rather than merging with it, and `linuxctl delete mcpd user` always removes both files' entries together.

A `create` (or a `token_hash`-migrated legacy user) still needs, before it can actually make any call:
1. A matching OS account in the `Dockerfile` (`useradd -m -s /bin/bash <username>`) - `linuxctl create` reminds you of this, but doesn't do it for you, since the account only becomes real after an image rebuild.
2. Tool/resource grants in `mcp-sudo.yaml` (a fresh user starts with none - denies everything by default).
3. `scripts/deploy.sh` to rebuild and redeploy.
