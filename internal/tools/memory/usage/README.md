# memory/usage

Reports memory and swap use from /proc/meminfo. Read-only. `used` is total - free - (buffers + cached + reclaimable slab); `available` is the kernel's MemAvailable. Text output is a `free`-like table with Mem: and Swap: rows (bytes, or e.g. `1.8Gi` with `human_readable`); `detailed: true` returns the raw /proc/meminfo instead, but only for text output (it is ignored with `output_format: json`). JSON (also yaml/table/wide) returns an object (total, used, free, shared, buffCache, available, swap_total, swap_used, swap_free) in bytes; `human_readable` is ignored there. For per-process memory use `processes/top` or `processes/list`, for CPU load `cpu/load-average`.

## Standard Output Format
All tools in the Linux MCP Daemon natively support returning structured JSON output. This can be requested by passing the `output_format` parameter.

## Parameters
- `output_format` (string, optional): The requested format. Setting this to `json`, `yaml`, `table`, or `wide` will return the raw structured JSON payload instead of human-readable text.
