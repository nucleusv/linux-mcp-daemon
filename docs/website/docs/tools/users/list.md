# list

**Tool Name**: `users/list`

Lists user accounts from `/etc/passwd` (uid, gid, home, shell, group memberships). Never reads `/etc/shadow` - this reports account identity, not credentials.

When `mcpd` runs containerized, passing `privileged: true` automatically lists the real host's users instead of the daemon's own container's - see [Master Daemon Configuration](../../configuration/daemon.md)'s `worker.containerized` setting.
