# `get_disk_usage`

This package implements the `get_disk_usage` tool (equivalent to `du`). It uses `filepath.WalkDir` to accurately calculate physical block sizes or apparent file sizes across the filesystem tree.

## Parameters
- `path` (string): The absolute path to start calculating from.
- `apparent_size` (boolean): Setting this to true forces it to check `info.Size()` for logical sizes instead of physical blocks.
- `threshold` (integer): A positive threshold skips files smaller than that byte size. A negative integer skips files larger than that size.
- `separate_dirs` (boolean): When true, isolates a directory's count so it only includes files directly inside it, skipping subdirectories.
- `max_depth` (integer): How deep to recurse (0 for summarize only).
- `one_file_system` (boolean): Skip directories on different file systems.
- `exclude` (array of strings): File name patterns to exclude.
- `all` (boolean): Write counts for all files, not just directories.
- `privileged` (boolean): Set to true to execute the worker as the root user.
