# process resource template

Handler for `process://{pid}/{target}`: one process's metadata, read from procfs by the `processes/read` worker.

| `{target}` | Returns |
|---|---|
| `status` | `/proc/<pid>/status` (name, state, uid/gid, memory, threads) |
| `cmdline` | the command line, NUL separators shown as spaces |
| `limits` | `/proc/<pid>/limits` (resource limits) |
| `environ` | the process environment - secret-shaped data, so `linuxctl describe processes` deliberately does not read it |

`{pid}` must be a number; anything else is refused. The result is `text/plain`.

## Permissions
The worker runs as the caller's OS account, so the kernel decides what of another user's process is visible (`environ` of a process you do not own is refused). As root only when the user's `resources:` grant for `process://` allows that pid (`resources: {"process://": ["*"]}`).

`linuxctl`: `describe processes 1234` (status, cmdline and limits together); to list processes and find a pid use `get processes`.
