# os://release

Static resource: the contents of `/etc/os-release`, verbatim (`text/plain`) - distribution name, version, ID, `VERSION_ID`, `PRETTY_NAME` and the rest of the `KEY=value` lines. Read by the daemon's resource handler, cached for 60 seconds.

For the kernel (name, release, machine) read `os://uname`; for a tool that returns the distribution and kernel version together use `system/os-release`.

Unprivileged: `/etc/os-release` is world-readable, so no grant is involved. `linuxctl get system release` (or `linuxctl resource os://release`).
