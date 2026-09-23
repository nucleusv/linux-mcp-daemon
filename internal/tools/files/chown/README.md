# files/chown

Changes owner and/or group, like `chown`, natively - no `chown` binary. Changing the owner needs `privileged: true`.

## Symlinks are never followed
Same guarantee as `files/chmod` (`internal/fsafe`): the path is walked component by component with `openat(O_PATH|O_NOFOLLOW)`, any symlink in it is refused, and ownership changes on exactly the verified object via `fchownat(fd, "", AT_EMPTY_PATH)`. With `recursive`, symlinks inside the tree are skipped and reported, never followed - so a root chown can't be redirected at `/etc/shadow` by a symlink planted in an allowed directory.

## Owner
`user`, `user:group`, `:group` (group only), `user:` (the user's login group); names or numeric ids. Names resolve from the `/etc/passwd` and `/etc/group` the worker sees (the host's, for privileged calls in containerized deployments).

## Parameters
- `path` (string, required): absolute path.
- `owner` (string, required): as above.
- `recursive` (boolean): also everything below a directory.
- `privileged` (boolean): run as root (authorized per path in `mcp-sudo.yaml`).

## Output
One line per change, `path: root:root -> www-data:www-data`; recursive runs end with a changed/unchanged/skipped summary. A no-op says `unchanged`.
