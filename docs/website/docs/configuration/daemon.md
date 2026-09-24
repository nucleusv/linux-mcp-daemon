---
sidebar_position: 1
---

# Master Daemon Configuration

The core settings for the Linux MCP Daemon are defined in `daemon.yaml`, in the config directory: `/etc/mcpd/configs` for both the systemd install and the container image. mcpd takes it from `--config-dir DIR`, else `$MCPD_CONFIG_DIR` (set in the image), else `configs/` under its working directory (the systemd unit runs in `/etc/mcpd`). `linuxctl`'s `mcpd` commands default their `--config-path` to `$MCPD_CONFIG_DIR` as well.

This file controls the global web server configuration, connection timeouts and rate limiting. The config directory holds three files:

| File | What it holds |
|---|---|
| `daemon.yaml` | server, worker, rate limits, tool timeouts (this page) |
| [`users.yaml`](./users) | who may connect: usernames and salted token hashes (mode `0600`) |
| [`mcp-sudo.yaml`](./mcp-sudo) | what each user may do as root |

## Example `daemon.yaml`

```yaml
server:
  tls:
    enabled: true
    port: 9091
    cert_file: tls/mcpd.crt      # relative to the config directory
    key_file: tls/mcpd.key
    generate: true               # self-signed pair on first start if missing
    # hosts: [mcp.example.com]   # extra names for a generated certificate
  http:
    enabled: false               # plain HTTP - tokens in clear text
    port: 9090

worker:
  timeout_seconds: 30
  containerized: true

rate_limits:
  default_rps: 5
  default_burst: 10

tools:
  disks/usage:
    timeout_seconds: 120
```

## Settings breakdown

- **`server.tls`** (on by default): HTTPS on `tls.port` (9091) with `cert_file`/`key_file` (relative paths are in the config directory). With `generate: true`, mcpd creates a self-signed ECDSA certificate there on its first start if neither file exists - covering the host name, `localhost`, every address of the machine and `tls.hosts` - valid 5 years, and logs its SHA-256 fingerprint (also `linuxctl describe mcpd tls`). Clients pin that fingerprint (`MCP_TLS_FINGERPRINT`) or trust the file (`MCP_CA_CERT`, `NODE_EXTRA_CA_CERTS`) - see [AI Agent Configuration](../ai-agent-configuration). To use a CA-issued certificate, put its files at those paths. To replace a generated one, delete both files and restart.
- **`server.http`** (off by default): plain HTTP on `http.port` (9090). Bearer tokens then cross the network in clear text - only for a trusted network, or behind a proxy that terminates TLS. mcpd logs a warning whenever it serves plain HTTP.
- **`server.port`**: the setting of configs from before v0.3.3, which have no `server.http` block - they keep serving plain HTTP on `server.port` (and HTTPS on `tls.port`, default 9443, if `tls.enabled`), with the warning. Move to the `tls`/`http` blocks above; the two layouts can't be mixed.
- **`worker.timeout_seconds`**: The global maximum time an Ephemeral Worker is allowed to run before the Master daemon sends a `SIGKILL`. This prevents runaway processes.
- **`worker.containerized`**: Set to `true` when `mcpd` itself runs inside a container (e.g. Kubernetes, Docker) with its own private root filesystem, as this daemon's own deployment does (see `k8s/deployment.yaml`). When true, every `privileged: true` tool call also joins the real host's mount namespace before running, so tools like `system/packages` or `services/manage` administer the actual host rather than the daemon's own container image. Set `false` when `mcpd` runs directly on the host with no container boundary to cross - `mcpd` also self-checks this at startup and logs a warning if the configured value doesn't match what it detects about its own environment.
- **`rate_limits`**: Global rate limits applied to every authenticated user to prevent an AI from spamming the server and saturating your I/O.
- **`tools.<name>.timeout_seconds`**: Tool-specific overrides, keyed by the tool's full `<group>/<command>` name. Heavy tools like `disks/usage` can be granted longer execution windows than lightweight tools.
- **`users`**: no longer here - users and tokens live in [`users.yaml`](./users). Configs from before it that still list `users:` in `daemon.yaml` keep working (mcpd logs a warning); `linuxctl` moves the list to `users.yaml` on its next user change. Users in both files is an error.

## Logging

```yaml
logging:
  level: info        # error | warn | info | debug
  format: text       # text | json
  access_log: true   # one line per HTTP request, at any level
```

mcpd logs to stderr (the systemd journal, `docker logs`, `kubectl logs`) with Go's `log/slog`, one line per event. `format: text` puts time, level and message first, then the fields as `key=value`:

```text
2026-09-24T10:41:26.138Z INFO  tool call user=testuser session=6471f456fbb2aa86-1 tool=system/os-release privileged=false duration_ms=4 ok=true args={}
2026-09-24T10:41:26.147Z WARN  tool call denied user=testuser session=7c63513180d95a29-2 tool=disks/usage privileged=true duration_ms=0 ok=false args="{\"path\":\"/root\",\"privileged\":true}" error="Permission denied. Hint: You are not authorized to use 'privileged: true' for this tool in mcp-sudo.yaml"
2026-09-24T10:41:28.180Z INFO  config reloaded audit=true user=privileged changes=0 detail=""
2026-09-24T10:41:30.249Z ERROR cannot serve HTTP addr=:9091 err="listen tcp :9091: bind: address already in use"
```

`format: json` writes one JSON object per line with every field keyed (`time`, `level`, `msg`, ...) - use it when journald, Loki or ELK should pick the fields apart:

```json
{"time":"2026-09-24T10:41:26.138Z","level":"INFO","msg":"tool call","user":"testuser","session":"6471f456fbb2aa86-1","tool":"system/os-release","privileged":false,"duration_ms":4,"ok":true,"args":"{}"}
```

| Level | What it adds |
|---|---|
| `error` | failures to start or listen |
| `warn` | denied privileged calls, rate limiting, a session used by another user, UID-pin mismatches, rejected config reloads, startup warnings |
| `info` (default) | startup, one line per tool call (`user`, `session`, `tool`, `privileged`, `duration_ms`, `ok`, redacted `args`, `error`), config reloads, UID pins |
| `debug` | every JSON-RPC request and response (method, id, size), SSE sessions opening and closing |

Two kinds of lines are written at **any** level:
- **audit** (`audit=true`): every call that changes the host - `files/create`, `files/update`, `files/chmod`, `files/chown`, `processes/delete`, `services/manage`, `kernel/system-control` with a value - and every config reload with its list of changes. Turning detail down never hides who changed what.
- **access**: one line per HTTP request (`INFO  access client=... user=... method=... uri=... status=... duration_ms=...`), while `access_log` is on.

Nothing is logged that could leak: arguments are redacted (`content`, `token`, `value`, headers, ...), tool output is never logged (responses by size only, at debug), and tokens appear nowhere. The level, format and access log can be changed without a restart - edit `daemon.yaml` and reload (see below).

## Applying changes

mcpd reads its config files at startup. To apply changes to a running daemon without a restart, call the [`daemon/reload-config`](../mcp-api/tools/daemon/reload-config) tool - `linuxctl reload daemon`, or let `linuxctl` do it: every `linuxctl create|update|delete mcpd user` and `linuxctl edit mcpd config` ends with a reload. It re-reads all three files and applies everything except `server.*` and `worker.containerized`, which need a restart (the reload says so when they changed).

Files are validated before anything changes: if one is invalid, the daemon keeps running on the config it has. Validation is **strict** - a key mcpd doesn't know (`path:` instead of `paths:`, `alowed:`) is an error, not a silently ignored line. At startup unknown keys are only warned about, so an upgrade never stops the daemon over a key an older version accepted.

To edit a file safely, use `linuxctl edit mcpd config daemon|users|sudo` - like `visudo`, it opens a copy in `$VISUAL`/`$EDITOR`, checks it with the same strict parser when you save, and replaces the real file only if it passes (otherwise: edit again, or discard).
