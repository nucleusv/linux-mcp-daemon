# packages

**Tool Name**: `system/packages`

Lists installed packages, auto-detecting the package manager (`dpkg`/Debian/Ubuntu, `apk`/Alpine) by natively parsing its database file. RPM-based systems aren't supported natively yet.

When `mcpd` runs containerized, passing `privileged: true` automatically queries the real host's installed packages instead of the daemon's own container image - see [Master Daemon Configuration](../../configuration/daemon.md)'s `worker.containerized` setting and [Sudo Privileges](../../configuration/mcp-sudo.md) for authorizing `privileged` access.
