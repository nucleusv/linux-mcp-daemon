# `get_disk_space`

This package implements the `get_disk_space` tool (equivalent to `df`). It uses `syscall.Statfs` to calculate available disk space and inodes without shelling out.

## Parameters
- `path` (string): The absolute path to check.
- `inodes` (boolean): If set to true, it will query `stat.Files` and `stat.Ffree` to return the total and free inode counts for the partition, rather than byte sizes.
- `human_readable` (boolean): If set to true, the raw byte counts are formatted into human-readable strings (e.g. `24.5 GiB`).
- `privileged` (boolean): Set to true to execute the worker as the root user (if authorized).
