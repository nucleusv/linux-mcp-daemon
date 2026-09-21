---
sidebar_position: 2
---

# get_disk_free

This package implements the `get_disk_free` tool (equivalent to the native Unix `df` command). It uses `syscall.Statfs` to calculate available disk space and inodes instantly without shelling out.

## Parameters
- `path` (string): The absolute directory or mount point to check.
- `inodes` (boolean): If set to true, it will query `stat.Files` and `stat.Ffree` to return the total and free inode index counts for the partition, rather than byte sizes.
- `human_readable` (boolean): If set to true, the raw byte counts are formatted into human-readable strings (e.g., `24.5 GiB`).
- `privileged` (boolean): Set to true to execute the worker as the root user.

### Example JSON-RPC
```json
{
  "method": "tools/call",
  "params": {
    "name": "get_disk_free",
    "arguments": {
      "path": "/",
      "inodes": true,
      "human_readable": true
    }
  }
}
```
