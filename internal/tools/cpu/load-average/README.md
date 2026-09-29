# cpu/load-average

Returns the 1, 5 and 15 minute load averages from sysinfo(2). Read-only; the only parameter is `output_format`. Load counts runnable plus uninterruptible tasks, not CPU percent: compare it with the number of logical CPUs from `cpu/list`. Text is `Load Average: 0.52, 0.48, 0.45`; `output_format: json` (also yaml/table/wide) returns an object with numbers `1_min`, `5_min`, `15_min`. For per-process CPU use `processes/top`, for memory `memory/usage`.

## Standard Output Format
All tools in the Linux MCP Daemon natively support returning structured JSON output. This can be requested by passing the `output_format` parameter.

## Parameters
- `output_format` (string, optional): The requested format. Setting this to `json`, `yaml`, `table`, or `wide` will return the raw structured JSON payload instead of human-readable text.
