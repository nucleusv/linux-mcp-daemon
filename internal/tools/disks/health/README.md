# disks/health

SMART health data for a drive - `smartctl -j -a /dev/<device>`. This is one of the project's documented exceptions to kernel-first: reading SMART means ATA/NVMe/SCSI pass-through commands with many vendor quirks, so the external `smartctl` binary (smartmontools) is used, returning its JSON as-is.

## Requirements
- **Root:** reading SMART data needs `privileged: true`, authorized for `disks/health` in `mcp-sudo.yaml`. Without it `smartctl` gets "Permission denied" opening the device.
- **smartmontools installed on the host:** in containerized deployments, a privileged call joins the host's mount namespace, so `smartctl` is looked up on the host, not in this daemon's image. If it's missing, the tool says so plainly.
- **Physical (or pass-through) disks:** virtual disks such as QEMU/virtio usually report that SMART is unavailable - that comes back in smartctl's own JSON (`smart_support`, `messages`).

## Parameters
- `device` (string, required): a block device name like `sda`, `nvme0n1` or `sg1` - validated as a bare name, never a path, so it can't point outside `/dev`.
- `privileged` (boolean): run as root - required.

## Output
smartctl's JSON output (`smartctl -j`). A non-zero smartctl exit status is not treated as an error by itself: smartctl's exit code is a bitmask of drive conditions, so a failing drive still returns its full report.
