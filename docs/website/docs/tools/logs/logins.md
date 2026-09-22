# logins

**Tool Name**: `logs/logins`

Lists login history (wraps `last`) or failed login attempts (`type: "failed"`, wraps `lastb`). Returns raw text, not JSON - `last`/`lastb`'s output isn't safe to hand-parse into structured data reliably.

`type: "failed"` typically requires `privileged: true`, since `btmp` is usually root-only readable. When `mcpd` runs containerized, `privileged: true` also automatically reads the real host's login history - see [Master Daemon Configuration](../../configuration/daemon.md)'s `worker.containerized` setting.
