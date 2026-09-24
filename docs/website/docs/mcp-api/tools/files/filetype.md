# Filetype

**Tool Name**: `files/filetype`

Determines a file's MIME type - the answer `file -b --mime-type` gives - detected natively from the file's first 8 KiB, so it needs no `file(1)` on the host. Use `files/stat` for size, permissions and ownership instead. Also available as the `file://{path}/type` resource.

- Directories, devices, FIFOs, sockets and empty files get `file`'s `inode/*` types; a **symlink is reported as `inode/symlink`, not followed** (as `file` does).
- ELF binaries: `application/x-executable`, `application/x-pie-executable`, `application/x-sharedlib`, `application/x-object`, `application/x-coredump`.
- Archives and compressors (gzip, bzip2, xz, zstd, 7z, lz4, tar, ar), `.deb`, `.rpm`, SQLite, PDF, images, audio/video and fonts the Go standard library knows, WebAssembly; `#!` scripts by interpreter (`text/x-shellscript`, `text/x-script.python`, `text/x-perl`, ...); JSON; otherwise `text/plain` or `application/octet-stream`.

`file`'s libmagic knows far more formats (C source, makefiles, office documents, ...); for those this answers the generic `text/plain` or `application/octet-stream`. On 209 system binaries, libraries, configs and compressed docs of Ubuntu 24.04 (amd64 and arm64) the answers are identical to `file`'s.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

Output captured live on an Ubuntu 24.04 host - every answer below matches `file -b --mime-type` there.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get files filetype /usr/bin/ls
```

Output:
```text
application/x-pie-executable
```

More answers from the same host:
```text
/etc/hosts                                text/plain
/usr/bin/ls                               application/x-pie-executable
/usr/lib/x86_64-linux-gnu/libc.so.6       application/x-sharedlib
/usr/bin/lsb_release                      text/x-shellscript
/usr/share/doc/bash/changelog.Debian.gz   application/gzip
/etc/ssl/certs                            inode/directory
/usr/bin/python3                          inode/symlink
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "files/filetype", "arguments": {"path": "/usr/bin/ls"}}}'

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
        "text": "application/x-pie-executable\n"
      }
    ]
  }
}
```

</details>
