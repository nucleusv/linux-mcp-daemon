# timezone

Backs the `system://timezone` resource. Reports the configured IANA timezone (e.g. `America/New_York`) plus the current local offset/abbreviation and time.

## How it's determined
Prefers `/etc/timezone` (Debian/Ubuntu). Falls back to resolving the `/etc/localtime` symlink target and extracting the zone name after `zoneinfo/` (portable to distros without `/etc/timezone`, e.g. RHEL/Alpine). If neither exists (seen in practice on minimal images with no `tzdata` installed at all), reports `UTC` - not a guess, but the documented POSIX/glibc default when no timezone is configured.

## Architecture note
Read directly in the master daemon process (no `worker.SpawnWorker`), same narrow exception as its sibling `system://hostname` - see that package's README for why. Always reports this container's own timezone when `mcpd` runs containerized, never the real host's.

## Permissions
No privilege required.
