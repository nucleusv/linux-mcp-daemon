# journal-control

**Tool Name**: `logs/journal-control`

Queries the systemd journal (`journalctl` equivalent).

## Parameters

- `unit` (optional) - filter by systemd unit (e.g. `kubelet.service`).
- `lines` (optional) - number of most recent lines to return. Defaults to 100.
- `since` / `until` (optional) - time range filters (e.g. `"1 hour ago"`, `"yesterday"`, `"12:00"`).
- `reverse` (optional) - newest entries first.
- `boot` (optional) - restrict to the current boot (`journalctl -b`).
- `boot_offset` (optional) - select a prior boot relative to the current one, e.g. `-1` for the previous boot; implies `boot`.
- `output_format` (optional) - `json` for structured journal entries; omit for plain text.
- `privileged` (optional, **required in containerized deployments**) - run as root and join the host mount namespace. `journalctl` and the journal it reads are architecturally host-only and are never bundled into this daemon's own container image; without `privileged: true` this call fails with `journalctl: executable file not found in $PATH`.

## Example

```bash
$ linuxctl logs journal-control --unit kubelet.service --lines 3 --privileged true
Sep 22 23:03:19 desktop-control-plane kubelet[268]: I0922 23:03:19.328923 ...
Sep 22 23:03:15 desktop-control-plane kubelet[268]: I0922 23:03:15.316812 ...
```
