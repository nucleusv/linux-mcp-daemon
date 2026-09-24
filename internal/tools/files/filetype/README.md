# files/filetype

Determines a file's MIME type - the answer `file -b --mime-type` gives - natively, without running `file(1)`. Also serves the `file://{path}/type` resource template.

## How it works

1. `lstat`: directories, symlinks (not followed, like `file`), devices, FIFOs, sockets and empty files get `file`'s `inode/*` types.
2. ELF binaries are told apart with `debug/elf`: `application/x-executable`, `application/x-pie-executable` (the `DF_1_PIE` flag - `file`'s rule; `libc.so.6` has an interpreter but is a library), `application/x-sharedlib`, `application/x-object`, `application/x-coredump`.
3. Signatures in the first 8 KiB: archives and compressors (gzip, bzip2, xz, zstd, 7z, lz4, tar, ar), `.deb`, `.rpm`, SQLite, PDF, images, WebAssembly, PEM/PGP text - then the standard library's `http.DetectContentType` for the formats it knows.
4. `#!` scripts by interpreter (`text/x-shellscript`, `text/x-script.python`, `text/x-perl`, ...).
5. Otherwise `text/plain` for NUL-free UTF-8, `application/json` for small valid JSON, else `application/octet-stream`.

`file`'s libmagic knows far more formats (C source, makefiles, office documents, ...); for those this reports the generic `text/plain` or `application/octet-stream`. `filetype_test.go` compares with `file(1)` on a host's system binaries, libraries, configs and compressed docs where `file` is installed - 209 files, no differences on Ubuntu 24.04 amd64 and arm64.

## Permissions

Reads the first 8 KiB of the file as the calling user, or as root with `privileged: true` when granted in `mcp-sudo.yaml` (limited to the grant's `paths:`).
