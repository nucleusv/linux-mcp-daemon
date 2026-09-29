# cpu/list

Lists the machine's CPUs from /proc/cpuinfo. Read-only. Text output shows the number of logical processors and, for the FIRST processor only, vendor, model name, MHz (or BogoMIPS) and cache size. `output_format: json` (also yaml/table/wide) returns an array with one object per logical CPU using /proc/cpuinfo's own field names, which differ by architecture (x86 `model name`, `cpu MHz`, `flags`; ARM `CPU implementer`). It does not report sockets, cores or threads separately, and `topology_only` has no effect. For current load use `cpu/load-average`, for per-process CPU `processes/top`, for OS and kernel `system/os-release`.

## Standard Output Format
All tools in the Linux MCP Daemon natively support returning structured JSON output. This can be requested by passing the `output_format` parameter.

## Parameters
- `output_format` (string, optional): The requested format. Setting this to `json`, `yaml`, `table`, or `wide` will return the raw structured JSON payload instead of human-readable text.
- `topology_only` (boolean, optional): accepted but has no effect.
