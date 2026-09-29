# disks resource template

Handler for `disks://{name}/stats`: the I/O counters of one block device (`sda`, `vda`, `nvme0n1`) as JSON, through the `disks/performance` worker (`/proc/diskstats`).

The numbers are cumulative since boot, not rates: sample twice and subtract for throughput. An unknown device name returns an error; for every device at once use the `disks/performance` tool, for the device tree `disks/list`.

## Permissions
Unprivileged: `/proc/diskstats` is world-readable, so the worker always runs as the caller and no grant is involved.

`linuxctl`: `describe disks vda`.
