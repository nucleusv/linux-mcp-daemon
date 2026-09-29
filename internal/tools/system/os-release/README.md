# system/os-release

Returns the Linux distribution and kernel version: the contents of /etc/os-release plus the uname line (system, host name, release, version, machine). Read-only. In a containerized daemon /etc/os-release is the container image's, not the host's. Text has an `OS Release Info:` block with the raw file and a `Kernel Info:` line; `output_format: json` (also yaml/table/wide) returns an object with `os_release` (the raw file text, not parsed into fields) and `kernel` (the uname line). For CPU details use `cpu/list`, for kernel parameters `kernel/system-control`.

## Standard Output Format
All tools in the Linux MCP Daemon natively support returning structured JSON output. This can be requested by passing the `output_format` parameter.

## Parameters
- `output_format` (string, optional): The requested format. Setting this to `json`, `yaml`, `table`, or `wide` will return the raw structured JSON payload instead of human-readable text.
