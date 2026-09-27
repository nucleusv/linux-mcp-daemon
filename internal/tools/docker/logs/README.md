# docker/logs

Reads one container's logs (`GET /containers/<id>/logs`) — the shape `logs/journal-control` has for systemd units. Both streams interleaved in the order Docker recorded them, last 100 lines unless `lines` says otherwise, and never a follow: it returns what is there and ends.

A non-TTY container's log stream is 8-byte-framed per record (stream id + length); this package demuxes those frames, so `stdout: false` / `stderr: false` really drop one stream. A TTY container's stream is unframed and passed through as-is.

Answers are capped at 1 MiB — ask for a smaller tail rather than a bigger answer.

## Parameters
- `container` (string, required): name, full ID or ID prefix.
- `lines` (integer, optional): tail this many lines (default 100).
- `since` / `until` (string, optional): unix timestamp or RFC3339.
- `timestamps` (boolean, optional): prefix each line with Docker's own timestamp.
- `stdout` / `stderr` (boolean, optional): default both true.
- `output_format` (string, optional): `json`; default raw log text.

## Usage & Permissions
Root or nothing: needs `docker/logs: {allowed: true, containers: [...]}` in the caller's grant in `configs/mcp-sudo.yaml`. No `containers:` list refuses everything. Logs routinely contain secrets a container printed at startup, which is why this is granted separately from `docker/containers`.

`linuxctl logs docker <name>` calls this tool.
