---
sidebar_position: 3
---

# get_disk_usage

This package implements the `get_disk_usage` tool (equivalent to the native Unix `du` command). It uses `filepath.WalkDir` to accurately calculate physical block sizes or apparent file sizes across massive filesystem trees.

## Caching
Because this tool can cause extreme I/O saturation if asked to traverse `/` multiple times, it is heavily protected by our **Singleflight Caching** architecture in the Master daemon.

## Parameters
- `path` (string): The absolute path to start calculating from.
- `apparent_size` (boolean): Setting this to true forces it to check `info.Size()` for logical sizes instead of physical blocks on disk.
- `threshold` (integer): A positive threshold skips files smaller than that byte size. A negative integer skips files larger than that size.
- `separate_dirs` (boolean): When true, isolates a directory's count so it only includes files directly inside it, skipping subdirectories.
- `max_depth` (integer): How deep to recurse (0 for summarize only).
- `one_file_system` (boolean): Skip directories on different file systems.
- `exclude` (array of strings): File name patterns to exclude during traversal.
- `all` (boolean): Write counts for all files, not just directories.
- `privileged` (boolean): Set to true to execute the worker as the root user.

### Example JSON-RPC
```json
{
  "method": "tools/call",
  "params": {
    "name": "get_disk_usage",
    "arguments": {
      "path": "/home",
      "threshold": 104857600,
      "separate_dirs": true
    }
  }
}
```
