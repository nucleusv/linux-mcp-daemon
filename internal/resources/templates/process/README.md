# process resource template

Handler for `process://{pid}/{target}`: one process's metadata, read from procfs by the `processes/read` worker.

| `{target}` | Returns |
|---|---|
| `status` | `/proc/<pid>/status` as text (name, state, uid/gid, memory, threads) |
| `cmdline` | JSON array of the command-line arguments |
| `limits` | JSON array of the resource limits (`/proc/<pid>/limits`: name, soft, hard, units) |
| `open_files` | JSON array of `{fd, target}` for `/proc/<pid>/fd`, sorted by descriptor (`socket:[...]`, `pipe:[...]` for non-files) |
| `environ` | the process environment as parsed `KEY=VALUE` entries - secret-shaped data, so `linuxctl describe processes` deliberately does not read it |

`{pid}` must be a number; anything else is refused, and so is an unknown target (`unsupported target`). The template answers as `text/plain`; the JSON above is the text.

## Permissions
The worker runs as the caller's OS account, so the kernel decides what of another user's process is visible. It runs as root only when the user's `resources:` grant for `process://` matches the pid by prefix; an empty prefix grants every pid (`resources: {"process://": [""]}` - a literal `"*"` does not work for prefix schemes), and `["12"]` matches pids that start with 12.

`linuxctl`: `describe processes 1234` (status, cmdline and limits together); to find a pid use `get processes`.
