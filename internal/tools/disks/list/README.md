# disks/list

Lists block devices as a tree, equivalent to `lsblk`, read natively from `/sys/class/block/`, `/proc/thread-self/mountinfo` and `/proc/swaps` - no `lsblk` dependency.

- Partitions nest under their disk (sysfs nesting, e.g. `.../block/sda/sda1`).
- Stacked devices (LVM, dm-crypt, multipath, md RAID) nest under the devices they're built on, via each lower device's `holders/` directory. A volume spanning several devices appears under each of them, as in `lsblk`.
- `NAME` uses the device-mapper name (`vg0-root`) where there is one; the kernel name (`dm-1`) is kept in the JSON `kname` field.
- `RM` for a partition is inherited from its disk - partitions have no `removable` file of their own.
- `TYPE` follows lsblk: `disk`, `part`, `lvm`, `crypt`, `mpath`, `dm`, `raid*`, `loop`, `rom`.
- `MOUNTPOINTS` is matched by `major:minor` from mountinfo, plus `[SWAP]` for active swap partitions.
- Like `lsblk`, empty devices and RAM disks (major 1) are hidden unless `all: true`.

## Standard Output Format
Text output reproduces `lsblk`'s default layout:

```text
NAME            MAJ:MIN RM  SIZE RO TYPE MOUNTPOINTS
sda               8:0    0   60G  0 disk
├─sda1            8:1    0    2G  0 part /boot
└─sda2            8:2    0   58G  0 part
  ├─vg4114-swap 252:0    0  3.7G  0 lvm  [SWAP]
  └─vg4114-root 252:1    0 54.3G  0 lvm  /
sr0              11:0    1    2K  1 rom
```

`json`/`yaml`/`table`/`wide` return the same tree as `{"blockdevices": [...]}` (the shape of `lsblk -J`), each node carrying `name`, `kname`, `maj:min`, `rm`, `size`, `size_bytes`, `ro`, `type`, `mountpoints` and, when present, `children`.

## Parameters
- `output_format` (string, optional): `json`, `yaml`, `table`, or `wide` return the structured tree instead of text.
- `all` (boolean, optional): Include empty devices and RAM disks (`lsblk -a`).
- `privileged` (boolean, optional): Run as root. In containerized deployments this reads the host's mount table, so `MOUNTPOINTS` reflects the host rather than the container.
