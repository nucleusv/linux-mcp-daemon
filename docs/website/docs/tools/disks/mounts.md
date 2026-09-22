# mounts

**Tool Name**: `disks/mounts`

Lists mounted filesystems (device, mount point, type, options) - equivalent to `mount`/`findmnt`'s basic view. Use disks/list for block devices instead.

When `mcpd` runs containerized, passing `privileged: true` automatically shows the real host's mount table instead of the daemon's own container's - see [Master Daemon Configuration](../../configuration/daemon.md)'s `worker.containerized` setting.
