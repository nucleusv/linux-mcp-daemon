# users/list

Lists user accounts by natively parsing `/etc/passwd` and `/etc/group` (uid, gid, home directory, shell, GECOS comment field, and both primary and supplementary group memberships).

## Security note
This tool **never reads `/etc/shadow`**. `/etc/passwd`/`/etc/group` are world-readable account metadata, not credentials - password hashes live in `/etc/shadow`, which this tool deliberately does not touch. There is no plan to add shadow-file access to this tool; a credential-reading feature would need its own explicit, separately-scoped design.

## Host vs. container filesystem
By default this reports on whatever filesystem the worker process sees. When `mcpd` runs containerized (`worker.containerized: true` in `configs/daemon.yaml`), passing `privileged: true` automatically also joins the host's real mount namespace, so this shows the real host's users instead of this daemon's own container's.

## Standard Output Format
All tools in the Linux MCP Daemon natively support returning structured JSON output. This can be requested by passing the `output_format` parameter.

## Parameters
- `output_format` (string, optional): The requested format. Setting this to `json`, `yaml`, `table`, or `wide` will return the raw structured JSON payload (array of `{username, uid, gid, group_name, comment, home_dir, shell, groups}`) instead of a human-readable summary line per user.
- `min_uid` (integer, optional): Only include users with UID >= this value (e.g. `1000` to exclude system/service accounts and see only "real" human users).
- `privileged` (boolean, optional): Run the worker as root. See "Host vs. container filesystem" above for what this means when `mcpd` runs containerized.
