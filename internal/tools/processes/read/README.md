# read

Internal worker behind the `process://{pid}/{target}` resource template. Not a callable tool (not in `tools.go`'s schema/`standardWorkers`) - reachable only via the resource URI. See `ARCHITECTURE.md`'s "Gotcha: privileged resource reads need TWO grants, not one" for why `processes/read` still needs an entry in `mcp-sudo.yaml`'s `tools:` block despite this.

## Targets

All read natively from `/proc/{pid}/`, no external commands:

- `status` - raw text from `/proc/{pid}/status`: memory (VmRSS, VmSize...), identity (Uid/Gid), threading, capabilities, scheduling state.
- `cmdline` - `/proc/{pid}/cmdline`, null-byte separated, returned as a JSON array (the process's argv).
- `environ` - `/proc/{pid}/environ`, null-byte separated `KEY=VALUE` pairs, returned as a JSON object.
- `limits` - `/proc/{pid}/limits`, a fixed-width table (rlimits). Parsed by splitting on runs of 2+ spaces rather than `strings.Fields`, since limit names themselves contain single spaces (e.g. "Max open files") which would otherwise misalign the columns. Returned as a JSON array of `{name, soft, hard, units}`.
- `open_files` - lists `/proc/{pid}/fd/`, resolving each entry's symlink target via `os.Readlink`. Targets are either real file paths, or pseudo-paths like `socket:[12345]`, `pipe:[12345]`, or `anon_inode:...` for non-file descriptors. Returned as a JSON array of `{fd, target}`, sorted numerically by fd.

## Permissions

Reading another user's `environ`, `limits`, or `open_files` typically requires root (the kernel restricts these for processes you don't own). Refer to `configs/mcp-sudo.yaml` for the privilege requirements.
