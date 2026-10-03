# List

**Tool Name**: `files/list`

Lists the entries of ONE directory like `ls -l` (dotfiles, `.` and `..` are hidden unless `all: true`): type and permissions, link count, owner, group, size, modification time and symlink targets (symlinks are shown, never followed). Read-only, not recursive, no entry cap; `path` must be absolute. For a recursive or filtered search use `files/find`, for a directory's total size `disks/usage`, for a file's MIME type `files/filetype`. On permission denied retry with `privileged: true` if granted (root calls also need a `paths:` entry covering the path). Text output is `ls -l` lines (`Directory is empty.` when empty; `long: false` gives names only, directories suffixed `/`). `output_format: json` returns an array of objects (name, type, mode, mode_octal, links, owner, group, uid, gid, size, modified, is_dir, target), `[]` when empty. Permission strings show setuid/setgid/sticky as `s`/`S`/`t`/`T`; device sizes appear as `major, minor`; the date shows the time for the last six months and the year otherwise.

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `path` | string | yes | Absolute path of the directory to list |
| `all` | boolean | no | Include dotfiles, . and .. (ls -a) |
| `dirs_first` | boolean | no | List directories before files |
| `human_readable` | boolean | no | Sizes like 4.0K, 1.5M (ls -h); default is bytes |
| `long` | boolean | no | Long listing like ls -l: type+permissions, links, owner, group, size, date, symlink target. Default true; false lists names only |
| `numeric_ids` | boolean | no | Show numeric uid/gid instead of names (ls -n) |
| `output_format` | string | no | Use json for structured output (yaml, table and wide return the same JSON); default is text |
| `privileged` | boolean | no | Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused |
| `reverse` | boolean | no | Reverse the sort order (ls -r) |
| `sort` | string | no | Sort by name (default), size (largest first) or time (newest first) One of: `name`, `size`, `time`. |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get files list /var/log --human_readable true
```

Output:
```text
# (first lines; Ubuntu 24.04)
total 4.9M
-rw-r--r-- 1 root    root              33K Sep 23 21:03 alternatives.log
-rw-r----- 1 root    adm                 0 Jun  6  2024 apport.log
drwxr-xr-x 2 root    root             4.0K Sep 23 21:02 apt
-rw-r----- 1 syslog  adm              924K Sep 24 02:42 auth.log
-rw------- 1 root    root              14K Sep 23 18:44 boot.log
-rw-r--r-- 1 root    root              60K Apr 23  2024 bootstrap.log
-rw-rw---- 1 root    utmp             1.5M Sep 24 02:42 btmp
drwxr-x--- 2 _chrony _chrony          4.0K Sep 23 18:42 chrony
-rw-r----- 1 root    adm              4.7K Jun  6  2024 cloud-init-output.log
-rw-r----- 1 root    adm               83K Jun  6  2024 cloud-init.log
drwxr-xr-x 2 root    root             4.0K Apr 19  2024 dist-upgrade
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
# 1. Open the SSE stream (in the background) and capture the one-time POST endpoint
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

# 2. POST the tools/call request to that endpoint
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "files/list", "arguments": {"path": "/var/log", "human_readable": true}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "total 4.9M\n-rw-r--r-- 1 root    root              33K Sep 23 21:03 alternatives.log\n-rw-r----- 1 root    adm                 0 Jun  6  2024 apport.log\ndrwxr-xr-x 2 root    root             4.0K Sep 23 21:02 apt\n-rw-r----- 1 syslog  adm              924K Sep 24 02:42 auth.log\n-rw------- 1 root    root              14K Sep 23 18:44 boot.log\n-rw-r--r-- 1 root    root              60K Apr 23  2024 bootstrap.log\n-rw-rw---- 1 root    utmp             1.5M Sep 24 02:42 btmp\ndrwxr-x--- 2 _chrony _chrony          4.0K Sep 23 18:42 chrony\n-rw-r----- 1 root    adm              4.7K Jun  6  2024 cloud-init-output.log\n-rw-r----- 1 root    adm               83K Jun  6  2024 cloud-init.log\ndrwxr-xr-x 2 root    root             4.0K Apr 19  2024 dist-upgrade\n"
      }
    ]
  }
}
```

</details>
