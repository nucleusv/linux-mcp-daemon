# files/list

Lists the entries of ONE directory like `ls -l` (dotfiles, `.` and `..` are hidden unless `all: true`): type and permissions, link count, owner, group, size, modification time and symlink targets (symlinks are shown, never followed). Read-only, not recursive, no entry cap; `path` must be absolute. For a recursive or filtered search use `files/find`, for a directory's total size `disks/usage`, for a file's MIME type `files/filetype`. On permission denied retry with `privileged: true` if granted (root calls also need a `paths:` entry covering the path). Text output is `ls -l` lines (`Directory is empty.` when empty; `long: false` gives names only, directories suffixed `/`). `output_format: json` returns an array of objects (name, type, mode, mode_octal, links, owner, group, uid, gid, size, modified, is_dir, target), `[]` when empty.

```text
total 24
drwxr-xr-x  2 root root 4096 Sep 23 18:39 ssh_config.d
-rw-------  1 root root  411 Sep 23 18:39 ssh_host_ed25519_key
lrwxrwxrwx  1 root root   24 Jun  6  2024 resolv.conf -> ../run/systemd/resolve/stub-resolv.conf
```

Type and permissions (with `s`/`S`/`t`/`T` for setuid/setgid/sticky), link count, owner, group, size (`major, minor` for devices), date (time for the last six months, year otherwise), name, and symlink target. Entries are `lstat`ed: a symlink is shown as a symlink with its target, never followed. The `total` line is in 1K blocks, as in ls. Output matches GNU `ls -la` column for column (`TestMatchesGNULs`).

## Parameters
- `path` (string, required): directory to list.
- `all` (boolean): include dotfiles, `.` and `..` (ls -a).
- `long` (boolean): long listing, default `true`; `false` lists names only (directories with a trailing `/`).
- `human_readable` (boolean): sizes like `4.0K`, `1.5M` (ls -h).
- `sort` (string): `name` (default; case-insensitive, leading dot ignored), `size` (largest first), `time` (newest first).
- `reverse` (boolean): reverse the sort (ls -r).
- `dirs_first` (boolean): directories before files.
- `numeric_ids` (boolean): numeric uid/gid (ls -n).
- `output_format` (string): `json`/`yaml`/`table`/`wide` return entries with `name`, `type`, `mode`, `mode_octal`, `links`, `owner`, `group`, `uid`, `gid`, `size`, `device`, `modified`, `is_dir`, `target`.
- `privileged` (boolean): run as root (authorized per path in `mcp-sudo.yaml`).

`all` and `long` were advertised by the schema before but never read by the code - hidden files were always shown and `long` did nothing.
