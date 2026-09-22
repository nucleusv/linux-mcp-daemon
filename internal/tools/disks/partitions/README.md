# disks/partitions

Retrieves partition boundaries for a block device - conceptually `fdisk -l`, but implemented by parsing `/sys/class/block` natively instead of wrapping the `fdisk` binary (this project avoids wrapping utilities where a native kernel interface exists; see `CLAUDE.md`).

## How it works

Every entry under `/sys/class/block` is either a whole disk or a partition. The kernel marks partitions by giving them (and only them) a `partition` file containing their partition number. For each partition found:

- `/sys/class/block/<name>/start` - start offset, in 512-byte sectors
- `/sys/class/block/<name>/size` - size, in 512-byte sectors
- the partition's parent disk is recovered by resolving the `/sys/class/block/<name>` symlink and taking the parent directory's name (e.g. `vda1` resolves into `.../block/vda/vda1`, giving parent `vda`) - this works uniformly across naming schemes (`sda1`, `vda1`, `nvme0n1p1`, `mmcblk0p1`) without guessing at name-prefix conventions.

## Parameters

- `device` (optional) - only return partitions belonging to this disk (e.g. `vda`). Omit to list partitions across every disk.
- `output_format` (optional) - `json` for structured output; omit for a plain text table.

## Usage & Permissions

No root required - `/sys/class/block` partition metadata is world-readable. Not present in `configs/mcp-sudo.yaml` as a privileged-only tool.
