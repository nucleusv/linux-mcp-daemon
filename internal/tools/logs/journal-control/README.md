# logs/journal-control

Queries the systemd journal - a `journalctl` equivalent, wrapping the `journalctl` binary directly (this is a documented exception to the project's kernel-first rule: the journal's binary format has no practical native-Go parser, matching the same reasoning as the `smartctl`/`traceroute` exceptions in `ARCHITECTURE.md`).

## Parameters

- `unit` - filter by systemd unit (e.g. `kubelet.service`).
- `lines` - number of most recent lines to return. Defaults to 100.
- `since` / `until` - time range filters (e.g. `"1 hour ago"`, `"yesterday"`, `"12:00"`).
- `reverse` - newest entries first.
- `boot` - restrict to the current boot (`journalctl -b`).
- `boot_offset` - select a prior boot relative to the current one (e.g. `-1` for the previous boot); implies `boot`.
- `output_format` - `json` for structured journal entries; omit for plain text.
- `privileged` - run as root and join the host mount namespace.

## Usage & Permissions

**`privileged: true` is required, not optional, in containerized deployments** (`worker.containerized: true` in `configs/daemon.yaml`) - `journalctl` and the systemd journal it reads are architecturally host-only and are never bundled into this daemon's own container image by design. Without it, the call fails with `journalctl: executable file not found in $PATH`, which is expected rather than a bug. See `configs/mcp-sudo.yaml` for which users are granted this.
