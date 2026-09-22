# locale

Backs the `system://locale` resource. Reports configured locale settings (`LANG`, `LC_*`).

## How it's determined
Prefers `/etc/default/locale` (Debian/Ubuntu), then `/etc/locale.conf` (systemd/RHEL style), whichever exists first. Falls back to this process's own `LANG`/`LC_ALL`/`LC_CTYPE` environment variables if neither file exists - a best-effort signal, not necessarily what an interactive login shell would see.

## Architecture note
Read directly in the master daemon process (no `worker.SpawnWorker`), same narrow exception as its sibling `system://hostname` - see that package's README for why. Always reports this container's own locale when `mcpd` runs containerized, never the real host's.

## Permissions
No privilege required.
