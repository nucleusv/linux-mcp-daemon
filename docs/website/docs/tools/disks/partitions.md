# partitions

**Tool Name**: `disks/partitions`

Retrieves partition boundaries for a block device (conceptually `fdisk -l`), parsed natively from `/sys/class/block` - this daemon does not wrap the `fdisk` binary. Returns each partition's device name, parent disk, partition number, and start/size in both sectors and bytes. No root required.

## Parameters

- `device` (optional) - only return partitions belonging to this disk (e.g. `vda`). Omit to list partitions across every disk on the system.
- `output_format` (optional) - `json` for structured output; omit for a plain text table.

## Example

```bash
$ linuxctl disks partitions --device vda --output json
[
  {
    "device": "vda1",
    "parent_disk": "vda",
    "number": 1,
    "start_sector": 2048,
    "size_sectors": 124997632,
    "size_bytes": 63998787584
  }
]
```
