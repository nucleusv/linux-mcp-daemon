# logs/logins

Lists login history (`last`) or failed login attempts (`type: "failed"`, `lastb`).

## Why this wraps a binary instead of parsing natively
`wtmp`/`btmp` are fixed-size binary C structs (`struct utmp`) whose exact field layout is glibc/arch-specific - unlike the plain-text formats this daemon otherwise hand-parses natively (dpkg's status file, `/proc/limits`, `/proc/mounts`), getting a binary struct's offsets/sizes wrong silently produces garbage instead of a clear error. This follows the project's documented precedent for wrapping a CLI (`smartctl`, `traceroute`, `file`) rather than hand-rolling something fragile. `last`/`lastb`/`who` ship as part of `util-linux`, already present in the base image.

## Output format
Returns **raw text always**, regardless of `output_format` - there's no `output_format` parameter on this tool. `last`/`lastb` have no reliable structured/JSON output mode, and their text format has enough edge cases (variable-width host field, "still logged in", "gone - no logout", reboot markers) that a custom parser risks silently-wrong structured data. Honest raw text beats that.

## Permissions
`type: "failed"` (`lastb`, reading `/var/log/btmp`) typically requires root - failed login attempts are sensitive and the file is usually `0660 root:utmp` or stricter. `type: "success"` (`last`, reading `/var/log/wtmp`) is often but not always world-readable.

## Host vs. container filesystem
When `mcpd` runs containerized (`worker.containerized: true`), passing `privileged: true` automatically also joins the host's real mount namespace, so this reads the real host's login history instead of this daemon's own container's (which normally has none - nothing logs in to a worker container via getty/sshd).

## Parameters
- `type` (string, optional): `"success"` (default, wraps `last`) or `"failed"` (wraps `lastb`).
- `limit` (integer, optional): Only return this many most recent entries (`-n`).
- `user` (string, optional): Only return entries for this username. Validated against a strict pattern before being passed to the wrapped binary, since a value starting with `-` could otherwise be misread as a flag.
- `privileged` (boolean, optional): Run the worker as root. See "Permissions" and "Host vs. container filesystem" above.
