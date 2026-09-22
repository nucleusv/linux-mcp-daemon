# disks/mounts

Lists mounted filesystems by natively parsing `/proc/thread-self/mounts` (the same content as `/etc/mtab`, equivalent to `mount`'s or `findmnt`'s basic view: device, mount point, filesystem type, options). Uses `/proc/thread-self` rather than `/proc/self` as a precaution around per-thread `setns` (see `ARCHITECTURE.md`) - not a confirmed bug fix, just removing a theoretical risk at no cost.

## Host vs. container filesystem
By default this reports on whatever mount namespace the worker process is in. When `mcpd` runs containerized (`worker.containerized: true` in `configs/daemon.yaml`), passing `privileged: true` automatically joins the host's real mount namespace first (see `internal/worker/hostns.go`), so this shows the real host's mounts instead of this daemon's own container's.

## Escaping
Device and mount point paths use the standard `/etc/mtab` octal-escape convention for whitespace and backslashes (e.g. a space becomes `\040`) - this is unescaped before returning results.

## Standard Output Format
All tools in the Linux MCP Daemon natively support returning structured JSON output. This can be requested by passing the `output_format` parameter.

## Parameters
- `output_format` (string, optional): The requested format. Setting this to `json`, `yaml`, `table`, or `wide` will return the raw structured JSON payload (array of `{device, mount_point, fs_type, options}`) instead of a human-readable summary line per mount.
- `fs_type` (string, optional): Only include mounts of this filesystem type (e.g. `ext4`, `overlay`, `tmpfs`).
- `privileged` (boolean, optional): Run the worker as root. See "Host vs. container filesystem" above for what this means when `mcpd` runs containerized.
