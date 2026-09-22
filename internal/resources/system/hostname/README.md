# hostname

Backs the `system://hostname` resource (moved here from `system://` naming's predecessor `os://hostname` - the URI scheme changed to `system://` since hostname is a configurable system-wide setting, not a fixed OS/kernel build fact like `os://uname`/`os://release`).

Returns the native hostname via `os.Hostname()`.

## Architecture note
This is read directly in the master daemon process (no `worker.SpawnWorker` involved), same as its siblings `os://uname` and `os://release`. That's a narrow, deliberate exception to `CLAUDE.md`'s "master daemon must never read /proc/sys directly" rule: `os.Hostname()` is trivial, non-sensitive, and involves no privilege escalation or arbitrary path traversal, so the risk this rule guards against doesn't apply here. It also means this resource always reports the daemon's own container's hostname, not the real host's, when `mcpd` runs containerized - unlike `disks/mounts`/`users/list`/etc., which are worker-routed and can reach the real host via `privileged: true`.

## Permissions
No privilege required to read the hostname.
