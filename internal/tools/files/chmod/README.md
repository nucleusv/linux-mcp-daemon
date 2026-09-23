# files/chmod

Changes permission bits, like `chmod`, natively - no `chmod` binary.

## Symlinks are never followed
The path is resolved one component at a time with `openat(O_PATH|O_NOFOLLOW)`, each step relative to the directory the previous step opened (`internal/fsafe`). A symlink anywhere in the path - including the target itself - is refused, and the mode is changed on exactly the object that was checked (through its descriptor), so a directory swapped for a symlink mid-operation can't redirect it. With `recursive`, children are opened relative to their verified parent; symlinks inside the tree are skipped and listed in the result, never followed.

This matters most for `privileged: true`: without it, a symlink planted in an allowed directory (`/tmp/x -> /etc/shadow`) would let a root chmod reach a file outside the path allowlist.

## Modes
- Octal: `644`, `0755`, `4755`. GNU chmod semantics on directories: a mode of up to four digits keeps existing setuid/setgid bits; five digits (`00755`) sets them exactly.
- Symbolic: `[ugoa][+-=][rwxXst]`, comma-separated - `u+x`, `go-w`, `a=r`, `+X`, `u+s`, `+t`, `ug=rw,o-rwx`. `X` adds execute only to directories and already-executable files.

Verified against GNU chmod on files and directories across many modes (`chmod_linux_test.go`).

## Parameters
- `path` (string, required): absolute path.
- `mode` (string, required): octal or symbolic mode.
- `recursive` (boolean): also everything below a directory.
- `privileged` (boolean): run as root (authorized per path in `mcp-sudo.yaml`).

## Output
One line per change, `path: 0644 (-rw-r--r--) -> 0755 (-rwxr-xr-x)`; recursive runs end with `changed N, unchanged M, skipped K symlink(s)` and any errors. A no-op says `unchanged`.
