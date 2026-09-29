# disks/usage

Measures how much disk space a directory tree uses (native walk, like `du`). Read-only. For the free space of a whole filesystem use `disks/free`; to find individual big files use `files/find` with `size`. By default the reply is one grand total; `max_depth: N` also lists directories up to N levels deep, largest first; `all: true` adds a per-file list (text output only). Sizes are allocated blocks unless `apparent_size: true`; hard links count once, symlinks are never followed. `exclude` patterns containing `/` match the full path (`/proc`, `/var/lib/*`), others the base name. Unreadable directories are skipped silently, so an unprivileged total can under-count: use `privileged: true` (a grant with a `paths:` entry). Results are cached for 60 s per user and arguments. The shipped config allows 300 s, otherwise the default is 30 s. Text ends with `Total size of PATH: N`; `output_format: json` returns an object (path, total_size, human_size with `human_readable`, directory_sizes as a path-to-bytes map only when `max_depth` > 0).

## Standard Output Format
All tools in the Linux MCP Daemon natively support returning structured JSON output. This can be requested by passing the `output_format` parameter.

## Parameters
- `output_format` (string, optional): The requested format. Setting this to `json`, `yaml`, `table`, or `wide` will return the raw structured JSON payload instead of human-readable text.
