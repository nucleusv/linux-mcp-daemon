# processes/list

Lists processes (PID, PPID, user, state, RSS, command) read from `/proc`. Read-only. Use it to find a PID, filter by `user` or one `pid`, or sort by memory. For the CPU/memory header and %CPU on every row use `processes/top`; for the process that owns a port `network/connections`; to signal a process `processes/delete`; for deep per-PID metrics the `process://<pid>/<target>` resource. Sorted by PID unless `sort_by` is `mem` (RSS, largest first) or `cpu` (samples for 0.5 s, so the call takes at least that long, and only then does `cpu_percent` appear); `limit` applies after sorting. Kernel threads appear as `[name]`. Command lines can contain secrets passed as arguments. Text output is a table; `output_format: json` returns an array of objects (pid, user, comm, state, ppid, rss_bytes, cmdline, cpu_percent). Sizes are bytes unless `human_readable`.

## Standard Output Format
All tools in the Linux MCP Daemon natively support returning structured JSON output. This can be requested by passing the `output_format` parameter.

## Parameters
- `output_format` (string, optional): The requested format. Setting this to `json`, `yaml`, `table`, or `wide` will return the raw structured JSON payload instead of human-readable text.
