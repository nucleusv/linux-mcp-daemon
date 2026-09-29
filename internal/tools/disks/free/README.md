# disks/free

Reports space on the ONE filesystem holding `path` (statfs, like `df` for a single path): total, used and free bytes and use percent (`human_readable` for GiB). Read-only; `path` must be absolute. It does not name the device or mount point: use `disks/mounts` for that, `disks/list` for block devices, `disks/usage` to see which folders use the space. There is no all-filesystems mode; call it once per mount point. `free` is what non-root users can use, and `used` is total minus that. `inodes: true` returns inode counts as plain text and ignores `output_format` and `human_readable`. Text output is three lines; `output_format: json` returns an object (path, total_bytes, used_bytes, free_bytes, use_percent, plus total_human, used_human, free_human with `human_readable`). `privileged: true` (a grant, and a `paths:` entry for root) only for paths you cannot stat.

## Standard Output Format
All tools in the Linux MCP Daemon natively support returning structured JSON output. This can be requested by passing the `output_format` parameter.

## Parameters
- `output_format` (string, optional): The requested format. Setting this to `json`, `yaml`, `table`, or `wide` will return the raw structured JSON payload instead of human-readable text.
