---
sidebar_label: 'Protocol Methods'
sidebar_position: 1.5
---

# Protocol Methods

Before an AI agent calls a single tool, it asks the daemon what it offers: `initialize`, then `tools/list`, `resources/list` and `resources/templates/list`. Their answers are what the agent sees of mcpd - the tool names, what each does and which arguments it takes - so this page shows them in full, captured live on an Ubuntu 24.04 host. They are sent like any other request, over the SSE session described in the [Overview](./overview); `linuxctl` reads them on every run (`linuxctl get mcp-api tools|resources|info` prints them).

## `initialize`

The first call of a session: the protocol version and what the server implements.

```bash
# POST to the endpoint the SSE stream gave you (see the Overview), then read the reply on the stream
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from the SSE stream>" \
  -H "Authorization: Bearer $MCP_TOKEN" -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "curl", "version": "1"}}}'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "capabilities": {
      "resources": {},
      "tools": {}
    },
    "protocolVersion": "2024-11-05",
    "serverInfo": {
      "name": "linux-mcp-daemon",
      "version": "0.5.0"
    }
  }
}
```

mcpd implements `tools` and `resources` (without subscriptions); `prompts`, `logging` and `completion` aren't declared - see [the `mcp-api` meta-group](../linuxctl/mcp-meta-group) for the details.

## `tools/list`

Every tool mcpd has, 48 of them - the same list for every user. One entry in full:

```bash
# POST to the endpoint the SSE stream gave you (see the Overview), then read the reply on the stream
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from the SSE stream>" \
  -H "Authorization: Bearer $MCP_TOKEN" -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/list"}'
```

```json
{
  "annotations": {
    "destructiveHint": false,
    "idempotentHint": true,
    "openWorldHint": false,
    "readOnlyHint": true,
    "title": "Filesystem free space"
  },
  "description": "Reports space on the ONE filesystem holding `path` (statfs, like `df` for a single path): total, used and free bytes and use percent (`human_readable` for GiB). Read-only; `path` must be absolute. It does not name the device or mount point: use `disks/mounts` for that, `disks/list` for block devices, `disks/usage` to see which folders use the space. There is no all-filesystems mode; call it once per mount point. `free` is what non-root users can use, and `used` is total minus that. `inodes: true` returns inode counts as plain text and ignores `output_format` and `human_readable`. Text output is three lines; `output_format: json` returns an object (path, total_bytes, used_bytes, free_bytes, use_percent, plus total_human, used_human, free_human with `human_readable`). `privileged: true` (a grant, and a `paths:` entry for root) only for paths you cannot stat.",
  "inputSchema": {
    "properties": {
      "human_readable": {
        "description": "Sizes like 53.2 GiB (df -h); default is bytes",
        "type": "boolean"
      },
      "inodes": {
        "description": "Report inode counts instead of block usage (-i); the reply is always plain text and ignores output_format and human_readable",
        "type": "boolean"
      },
      "output_format": {
        "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
        "type": "string"
      },
      "path": {
        "description": "Absolute path to check",
        "type": "string"
      },
      "privileged": {
        "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused",
        "type": "boolean"
      }
    },
    "required": [
      "path"
    ],
    "type": "object"
  },
  "linuxctl_verb": "free",
  "name": "disks/free",
  "tools_group": "disks"
}
```

| Field | Meaning |
|---|---|
| `name` | What `tools/call` takes: `<group>/<command>` |
| `description` | What the tool does, which tool to use instead for related questions, and - for a few tools, when the user holds a root grant - a note saying so |
| `annotations` | MCP hints for clients: `title`, `readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint` - what needs a confirmation |
| `inputSchema` | JSON Schema of the arguments; `required` lists the mandatory ones |
| `tools_group`, `linuxctl_verb` | mcpd's own additions, for `linuxctl`'s `<verb> <group>` grammar; MCP clients ignore them |

<details>
<summary><b>The whole response</b> (48 tools)</summary>

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "tools": [
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "List directory"
        },
        "description": "Lists the entries of ONE directory like `ls -l` (dotfiles, `.` and `..` are hidden unless `all: true`): type and permissions, link count, owner, group, size, modification time and symlink targets (symlinks are shown, never followed). Read-only, not recursive, no entry cap; `path` must be absolute. For a recursive or filtered search use `files/find`, for a directory's total size `disks/usage`, for a file's MIME type `files/filetype`. On permission denied retry with `privileged: true` if granted (root calls also need a `paths:` entry covering the path). Text output is `ls -l` lines (`Directory is empty.` when empty; `long: false` gives names only, directories suffixed `/`). `output_format: json` returns an array of objects (name, type, mode, mode_octal, links, owner, group, uid, gid, size, modified, is_dir, target), `[]` when empty.",
        "inputSchema": {
          "properties": {
            "all": {
              "description": "Include dotfiles, . and .. (ls -a)",
              "type": "boolean"
            },
            "dirs_first": {
              "description": "List directories before files",
              "type": "boolean"
            },
            "human_readable": {
              "description": "Sizes like 4.0K, 1.5M (ls -h); default is bytes",
              "type": "boolean"
            },
            "long": {
              "description": "Long listing like ls -l: type+permissions, links, owner, group, size, date, symlink target. Default true; false lists names only",
              "type": "boolean"
            },
            "numeric_ids": {
              "description": "Show numeric uid/gid instead of names (ls -n)",
              "type": "boolean"
            },
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "path": {
              "description": "Absolute path of the directory to list",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused",
              "type": "boolean"
            },
            "reverse": {
              "description": "Reverse the sort order (ls -r)",
              "type": "boolean"
            },
            "sort": {
              "description": "Sort by name (default), size (largest first) or time (newest first)",
              "enum": [
                "name",
                "size",
                "time"
              ],
              "type": "string"
            }
          },
          "required": [
            "path"
          ],
          "type": "object"
        },
        "linuxctl_verb": "list",
        "name": "files/list",
        "tools_group": "files"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Read file contents"
        },
        "description": "Reads a text file and returns its contents as raw text (no line numbers or metadata). Read-only; `path` must be absolute. To find a file use `files/find`, for size or mode `files/list`, to check whether it is binary `files/filetype` (binary files are refused with `cannot read binary file`). Selection: `start_line`/`end_line` (1-indexed, inclusive; `start_line` alone reads to the end, `end_line` alone starts at line 1) take precedence over the byte range `offset`/`limit`. With no selection, or `offset` without `limit`, at most 10240 bytes come back followed by a `[WARNING: File truncated ...]` line, so page large files with `start_line`/`end_line` or pass `limit`. There is no streaming: one call reads into memory, and a line over 64 KiB fails line mode. A `start_line` past the end is an error; an empty file returns an empty string. `privileged: true` (needs a grant, and a `paths:` entry for root) reads files your account cannot.",
        "inputSchema": {
          "properties": {
            "end_line": {
              "description": "Last line to return (inclusive); alone it starts at line 1. Ignored if below start_line",
              "type": "integer"
            },
            "limit": {
              "description": "Number of bytes to return (no upper cap). With no selection at all, the first 10240 bytes are returned",
              "type": "integer"
            },
            "offset": {
              "description": "Byte offset to start at; ignored when start_line/end_line is given. Without limit at most 10240 bytes are returned",
              "type": "integer"
            },
            "path": {
              "description": "Absolute path of the file to read",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused",
              "type": "boolean"
            },
            "start_line": {
              "description": "First line to return (1-indexed). With or without end_line it takes precedence over offset/limit; alone it reads to the end of the file",
              "type": "integer"
            }
          },
          "required": [
            "path"
          ],
          "type": "object"
        },
        "linuxctl_verb": "get",
        "name": "files/read",
        "tools_group": "files"
      },
      {
        "annotations": {
          "destructiveHint": true,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": false,
          "title": "Create or overwrite file"
        },
        "description": "Writes a whole file: creates it, or REPLACES the content of an existing one entirely. Mutating and not atomic. Missing parent directories are created (mode 0755); a new file gets mode 0644, an existing file keeps its mode. With `content` omitted or empty it only touches: creates an empty file or refreshes the modification time and never truncates an existing file. To change part of an existing file use `files/update` (append or replace lines); to check first whether a path exists use `files/list` or `files/find`; afterwards set mode or owner with `files/chmod`/`files/chown`. `content` is text and no trailing newline is added; `path` must be absolute. Writing where your account cannot needs `privileged: true` (a grant, and for root a `paths:` entry covering the path, which also refuses symlinks in the path; otherwise a symlink at the path is followed). Returns one line, `Successfully created and wrote to PATH` or `Successfully touched PATH`; failures are plain text such as `failed to write to file`.",
        "inputSchema": {
          "properties": {
            "content": {
              "description": "Text to write; replaces any existing content entirely, no newline is added. Omit or leave empty to only create an empty file or refresh its mtime (never truncates)",
              "type": "string"
            },
            "path": {
              "description": "Absolute path of the file; missing parent directories are created",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused",
              "type": "boolean"
            }
          },
          "required": [
            "path"
          ],
          "type": "object"
        },
        "linuxctl_verb": "create",
        "name": "files/create",
        "tools_group": "files"
      },
      {
        "annotations": {
          "destructiveHint": true,
          "idempotentHint": false,
          "openWorldHint": false,
          "readOnlyHint": false,
          "title": "Edit file (append or replace lines)"
        },
        "description": "Edits part of an EXISTING file in place: appends text or replaces an inclusive line range. Mutating and not atomic; the file keeps its mode and owner. To write a whole file use `files/create`; to see the lines first use `files/read` with `start_line`/`end_line`. With `append: true` exactly `content` is added at the end (no newline is added, include your own) and the file is created if missing, though not its parent directories; append wins over any line range. Otherwise `start_line` AND `end_line` are both required (1-indexed, `end_line` >= `start_line`), the file must exist, and those lines are replaced by `content` (one trailing newline of `content` is ignored; an `end_line` past the end is clamped; a `start_line` past the end adds `content` as a new last line). There is no insert or delete mode: replacing with empty `content` leaves one empty line. Returns `Successfully appended to PATH` or `Successfully updated lines A-B in PATH`, no diff. `path` must be absolute; `privileged: true` (a grant, and for root a `paths:` entry) edits files you cannot write.",
        "inputSchema": {
          "properties": {
            "append": {
              "description": "If true, append content at the end (creates the file if missing); takes precedence over start_line/end_line",
              "type": "boolean"
            },
            "content": {
              "description": "Text to append, or to replace the line range with (no newline is added when appending)",
              "type": "string"
            },
            "end_line": {
              "description": "Last line to replace (inclusive, >= start_line; past the end of the file is clamped)",
              "type": "integer"
            },
            "path": {
              "description": "Absolute path of the file to edit",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused",
              "type": "boolean"
            },
            "start_line": {
              "description": "First line to replace (1-indexed); required together with end_line unless append is true",
              "type": "integer"
            }
          },
          "required": [
            "path",
            "content"
          ],
          "type": "object"
        },
        "linuxctl_verb": "update",
        "name": "files/update",
        "tools_group": "files"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Find files"
        },
        "description": "Searches a directory tree (default `/`) by name, type, age or size, like `find`. Read-only; never follows symlinks; skips `/proc`, `/sys`, `/dev` and `/run` when the search starts at `/` or above them (not when `path` is inside one). Use `files/list` to see one known directory and `disks/usage` to see which folders take the space. All filters are ANDed and none is required. There is NO result cap: `path: /` without a filter lists every file on the host, so give `name`, `type` or `max_depth`; the 30 s worker timeout applies. Syntax: `name` is a case-sensitive glob on the base name only (`*.log`, not a path); `type` is one of `f d l b c p s` or a comma list (`f,d`); `mtime` in days: `+7` older than 7 days, `-1` within the last day, `7` exactly 7 days; `size`: `+100M` larger, `-10k` smaller (units b c w k M G, rounded up); `max_depth` 1 = direct children, omitted or 0 = unlimited. Unreadable directories are skipped silently. Text output: one `SIZE PATH` line per match, sorted by path (no size for directories; empty output = no match). `output_format: json` returns an array of objects (path, size in bytes, type).",
        "inputSchema": {
          "properties": {
            "human_readable": {
              "description": "Text output only: sizes like 1.5 KiB; default is bytes",
              "type": "boolean"
            },
            "max_depth": {
              "description": "Levels below path to descend (1 = direct children); omit or 0 for unlimited",
              "type": "integer"
            },
            "mtime": {
              "description": "Days since modification: '+7' older than 7 days, '-1' within the last day, '7' exactly 7 days",
              "type": "string"
            },
            "name": {
              "description": "Glob on the file's base name only, case-sensitive, e.g. '*.log' (not a path pattern)",
              "type": "string"
            },
            "output_format": {
              "description": "json (yaml, table and wide return the same JSON) gives an array of objects with path, size, type; default is text",
              "type": "string"
            },
            "path": {
              "description": "Absolute starting directory (default /). A large tree with no filter can take longer than the 30 s worker limit",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused",
              "type": "boolean"
            },
            "size": {
              "description": "'+100M' larger than 100 MiB, '-10k' smaller than 10 KiB; units b c w k M G, rounded up to the unit",
              "type": "string"
            },
            "type": {
              "description": "f file, d directory, l symlink, b, c, p, s; or a comma list such as 'f,d'",
              "type": "string"
            }
          },
          "required": [],
          "type": "object"
        },
        "linuxctl_verb": "find",
        "name": "files/find",
        "tools_group": "files"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Detect file MIME type"
        },
        "description": "Returns a file's MIME type, like `file -b --mime-type`, detected natively from its first 8 KiB (no file(1) needed); read-only, `path` must be absolute. A symlink is reported as `inode/symlink` (never followed); directories, devices, fifos and sockets as `inode/...`. Use it before `files/read`, which refuses binary files. For size, permissions or ownership use `files/list`; for contents `files/read`. The reply is always one plain-text line (no `output_format`). `privileged: true` (a grant, and for root a `paths:` entry) reads files your account cannot.",
        "inputSchema": {
          "properties": {
            "path": {
              "description": "Absolute path to the file",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused",
              "type": "boolean"
            }
          },
          "required": [
            "path"
          ],
          "type": "object"
        },
        "linuxctl_verb": "filetype",
        "name": "files/filetype",
        "tools_group": "files"
      },
      {
        "annotations": {
          "destructiveHint": true,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": false,
          "title": "Change file permissions"
        },
        "description": "Changes a file's or directory's permission bits (chmod). Mutating and idempotent; for owner or group use `files/chown`, to check the result `files/list`. Never follows symbolic links: a path containing a symlink in any component is refused, and recursive changes skip symlinks and report them. Numeric modes follow GNU chmod semantics (on directories a 4-digit mode keeps setuid/setgid; use 5 digits, e.g. 00755, to set them exactly); a bare `755` is octal. Changing a file you do not own needs `privileged: true` (a grant, and a `paths:` entry for root). Single change returns `PATH: 0644 (-rw-r--r--) -> 0755 (-rwxr-xr-x)`, or `... unchanged` if already set. A recursive run prints one line per changed entry, then `changed N, unchanged M` (plus skipped symlinks) and per-entry errors; it is not atomic, so partial success is possible.",
        "inputSchema": {
          "properties": {
            "mode": {
              "description": "Octal (644, 0755, 4755) or symbolic (u+x, go-w, a=r, +X, u+s, +t; comma-separated)",
              "type": "string"
            },
            "path": {
              "description": "Absolute path",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root - needed for files you don't own. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path",
              "type": "boolean"
            },
            "recursive": {
              "description": "Also apply to everything below a directory (symlinks are skipped, never followed)",
              "type": "boolean"
            }
          },
          "required": [
            "path",
            "mode"
          ],
          "type": "object"
        },
        "linuxctl_verb": "chmod",
        "name": "files/chmod",
        "tools_group": "files"
      },
      {
        "annotations": {
          "destructiveHint": true,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": false,
          "title": "Change file owner/group"
        },
        "description": "Changes a file's or directory's owner and/or group (chown). Mutating and idempotent; for permission bits use `files/chmod`, to check the result `files/list`. Never follows symbolic links: a path containing a symlink in any component is refused, and recursive changes skip symlinks and report them. Changing the owner requires `privileged: true` (a grant, and a `paths:` entry for root). `owner` is `user`, `user:group`, `:group` or `user:` (the user's login group), with names or numeric ids; names are looked up in the host's /etc/passwd and /etc/group and an unknown one fails (`no such user`). Returns `PATH: old -> new` as owner:group, or `unchanged`; a recursive run prints one line per changed entry then a summary with skipped symlinks and per-entry errors, and is not atomic.",
        "inputSchema": {
          "properties": {
            "owner": {
              "description": "user, user:group, :group, or user: (the user's login group); names or numeric ids",
              "type": "string"
            },
            "path": {
              "description": "Absolute path",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root - required to change ownership. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path",
              "type": "boolean"
            },
            "recursive": {
              "description": "Also apply to everything below a directory (symlinks are skipped, never followed)",
              "type": "boolean"
            }
          },
          "required": [
            "path",
            "owner"
          ],
          "type": "object"
        },
        "linuxctl_verb": "chown",
        "name": "files/chown",
        "tools_group": "files"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Filesystem free space"
        },
        "description": "Reports space on the ONE filesystem holding `path` (statfs, like `df` for a single path): total, used and free bytes and use percent (`human_readable` for GiB). Read-only; `path` must be absolute. It does not name the device or mount point: use `disks/mounts` for that, `disks/list` for block devices, `disks/usage` to see which folders use the space. There is no all-filesystems mode; call it once per mount point. `free` is what non-root users can use, and `used` is total minus that. `inodes: true` returns inode counts as plain text and ignores `output_format` and `human_readable`. Text output is three lines; `output_format: json` returns an object (path, total_bytes, used_bytes, free_bytes, use_percent, plus total_human, used_human, free_human with `human_readable`). `privileged: true` (a grant, and a `paths:` entry for root) only for paths you cannot stat.",
        "inputSchema": {
          "properties": {
            "human_readable": {
              "description": "Sizes like 53.2 GiB (df -h); default is bytes",
              "type": "boolean"
            },
            "inodes": {
              "description": "Report inode counts instead of block usage (-i); the reply is always plain text and ignores output_format and human_readable",
              "type": "boolean"
            },
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "path": {
              "description": "Absolute path to check",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused",
              "type": "boolean"
            }
          },
          "required": [
            "path"
          ],
          "type": "object"
        },
        "linuxctl_verb": "free",
        "name": "disks/free",
        "tools_group": "disks"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Directory disk usage"
        },
        "description": "Measures how much disk space a directory tree uses (native walk, like `du`). Read-only. For the free space of a whole filesystem use `disks/free`; to find individual big files use `files/find` with `size`. By default the reply is one grand total; `max_depth: N` also lists directories up to N levels deep, largest first; `all: true` adds a per-file list (text output only). Sizes are allocated blocks unless `apparent_size: true`; hard links count once, symlinks are never followed. `exclude` patterns containing `/` match the full path (`/proc`, `/var/lib/*`), others the base name. Unreadable directories are skipped silently, so an unprivileged total can under-count: use `privileged: true` (a grant with a `paths:` entry). Results are cached for 60 s per user and arguments. The shipped config allows 300 s, otherwise the default is 30 s. Text ends with `Total size of PATH: N`; `output_format: json` returns an object (path, total_size, human_size with `human_readable`, directory_sizes as a path-to-bytes map only when `max_depth` > 0).",
        "inputSchema": {
          "properties": {
            "all": {
              "description": "Also list every file, largest first (text output only)",
              "type": "boolean"
            },
            "apparent_size": {
              "description": "Report logical file sizes instead of allocated disk blocks",
              "type": "boolean"
            },
            "exclude": {
              "description": "Patterns to skip: one containing '/' matches the full path (e.g. '/proc', '/var/lib/*'), others the base name (e.g. '*.tmp')",
              "items": {
                "type": "string"
              },
              "type": "array"
            },
            "human_readable": {
              "description": "Sizes like du -h (4.0 KiB); default is bytes",
              "type": "boolean"
            },
            "max_depth": {
              "description": "0 or omitted: grand total only; N: also list directories up to N levels deep, largest first",
              "type": "integer"
            },
            "one_file_system": {
              "description": "Skip directories on different file systems (-x)",
              "type": "boolean"
            },
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "path": {
              "description": "Absolute path of the directory to measure",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml and a `paths:` entry covering the path, otherwise refused",
              "type": "boolean"
            },
            "separate_dirs": {
              "description": "For directories do not include size of subdirectories (-S)",
              "type": "boolean"
            },
            "threshold": {
              "description": "Bytes: a positive value hides entries smaller than this, a negative value hides larger ones (printed lines only)",
              "type": "integer"
            }
          },
          "required": [
            "path"
          ],
          "type": "object"
        },
        "linuxctl_verb": "usage",
        "name": "disks/usage",
        "tools_group": "disks"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Process snapshot (top)"
        },
        "description": "Snapshot like `top -b -n 1`, read from `/proc`: header (uptime, users, load average, task counts by state, CPU us/sy/ni/id/wa/hi/si/st, memory and swap) plus the process table (PID USER PR NI VIRT RES SHR S %CPU %MEM TIME+ COMMAND), sorted by `sort_by` (default cpu). Read-only. %CPU is measured over `interval_ms` (default 1000, max 10000), so the call takes about that long. For a lighter PID list or a single PID use `processes/list`; to signal a process `processes/delete`; for memory totals only `memory/usage`; for which process owns a port `network/connections`. `limit` defaults to all processes; `user` is an exact username. Sizes are bytes (MiB with `human_readable`). `output_format`: default and `table` give top's layout, `wide` adds PPID, THR and full command lines, `json`/`yaml` return an object with `summary` and `processes` (memory in bytes).",
        "inputSchema": {
          "properties": {
            "human_readable": {
              "description": "Memory like top (MiB header, m/g columns); default is bytes",
              "type": "boolean"
            },
            "interval_ms": {
              "description": "%CPU sampling interval in milliseconds (default 1000, max 10000)",
              "type": "integer"
            },
            "limit": {
              "description": "Maximum processes to list (default: all)",
              "type": "integer"
            },
            "output_format": {
              "description": "Default/table: top's own layout. wide: adds PPID, THR and full command lines (like top -c). json/yaml: structured {summary, processes}, memory in bytes (mem_bytes, swap_bytes, virt_bytes, res_bytes, shr_bytes)",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused",
              "type": "boolean"
            },
            "sort_by": {
              "description": "Sort column: cpu (default, like top), mem/res (resident memory), time (total CPU time), pid",
              "enum": [
                "cpu",
                "mem",
                "res",
                "time",
                "pid"
              ],
              "type": "string"
            },
            "user": {
              "description": "Only this user's processes",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "top",
        "name": "processes/top",
        "tools_group": "processes"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "List processes"
        },
        "description": "Lists processes (PID, PPID, user, state, RSS, command) read from `/proc`. Read-only. Use it to find a PID, filter by `user` or one `pid`, or sort by memory. For the CPU/memory header and %CPU on every row use `processes/top`; for the process that owns a port `network/connections`; to signal a process `processes/delete`; for deep per-PID metrics the `process://<pid>/<target>` resource. Sorted by PID unless `sort_by` is `mem` (RSS, largest first) or `cpu` (samples for 0.5 s, so the call takes at least that long, and only then does `cpu_percent` appear); `limit` applies after sorting. Kernel threads appear as `[name]`. Command lines can contain secrets passed as arguments. Text output is a table; `output_format: json` returns an array of objects (pid, user, comm, state, ppid, rss_bytes, cmdline, cpu_percent). Sizes are bytes unless `human_readable`.",
        "inputSchema": {
          "properties": {
            "human_readable": {
              "description": "RSS like 10Mi; default is bytes",
              "type": "boolean"
            },
            "limit": {
              "description": "Maximum rows returned, applied after sorting",
              "type": "integer"
            },
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "pid": {
              "description": "Filter to a single specific PID",
              "type": "integer"
            },
            "privileged": {
              "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused",
              "type": "boolean"
            },
            "sort_by": {
              "description": "pid (default), mem (RSS, largest first) or cpu (samples for 0.5 s)",
              "type": "string"
            },
            "user": {
              "description": "Exact username",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "get",
        "name": "processes/list",
        "tools_group": "processes"
      },
      {
        "annotations": {
          "destructiveHint": true,
          "idempotentHint": false,
          "openWorldHint": false,
          "readOnlyHint": false,
          "title": "Signal a process"
        },
        "description": "Sends ONE signal to ONE process by PID (kill(2)); the default SIGTERM asks the process to exit. Mutating and not idempotent: it returns as soon as the signal is delivered and does not check that the process exited, and signalling a PID that is gone fails with `no such process`. Allowed `signal` values: SIGTERM, SIGKILL, SIGHUP, SIGINT, SIGQUIT, SIGUSR1, SIGUSR2, SIGSTOP, SIGCONT, SIGABRT (SIG prefix optional, case-insensitive, numbers rejected), so it can also pause (SIGSTOP) and resume (SIGCONT). Refuses PID 1 and the mcpd daemon itself. Another user's process fails with `not permitted` unless `privileged: true` (needs a grant). Find the PID first with `processes/list` or `processes/top`. To stop a managed service use `services/manage` (systemd may restart a killed one), for a container `docker/manage`. Returns one text line, `Successfully sent signal SIGTERM to process N`; `output_format` has no effect.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "Ignored - the reply is always text",
              "type": "string"
            },
            "pid": {
              "description": "PID to signal (positive integer; PID 1 and mcpd itself are refused)",
              "type": "integer"
            },
            "privileged": {
              "description": "Run as root to signal other users' processes. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused",
              "type": "boolean"
            },
            "signal": {
              "description": "SIGTERM (default), SIGKILL, SIGHUP, SIGINT, SIGQUIT, SIGUSR1, SIGUSR2, SIGSTOP, SIGCONT or SIGABRT; SIG prefix optional, case-insensitive",
              "type": "string"
            }
          },
          "required": [
            "pid"
          ],
          "type": "object"
        },
        "linuxctl_verb": "delete",
        "name": "processes/delete",
        "tools_group": "processes"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": true,
          "readOnlyHint": true,
          "title": "DNS lookup"
        },
        "description": "Resolves DNS records for a hostname through the host's resolver (/etc/resolv.conf; a DNS server cannot be chosen), so answers may come from a local cache. Read-only, but it sends queries off-host. `record_type` (case-insensitive) is A, AAAA, CNAME, TXT, MX, NS or ANY; the default ANY is not a DNS ANY query but runs the CNAME, A/AAAA, TXT, MX and NS lookups in turn. Other types (SOA, PTR, SRV) are rejected, and `host` must be a name: an IP address is not reverse-resolved. Always returns JSON: an object with `host` and `records` (array of objects with `type` and `value`; an MX value looks like `10 mail.example.com.`). Empty `records` means the name exists but has no record of that type; a name that does not exist is an error (`HOST: no such host`). To test reachability use `network/ping`, to fetch a URL `network/curl`.",
        "inputSchema": {
          "properties": {
            "host": {
              "description": "Hostname to resolve, e.g. example.com (an IP address is not reverse-resolved)",
              "type": "string"
            },
            "record_type": {
              "description": "A, AAAA, CNAME, TXT, MX, NS or ANY (default ANY: all of these are looked up in turn)",
              "type": "string"
            }
          },
          "required": [
            "host"
          ],
          "type": "object"
        },
        "linuxctl_verb": "nslookup",
        "name": "network/nslookup",
        "tools_group": "network"
      },
      {
        "annotations": {
          "destructiveHint": true,
          "idempotentHint": false,
          "openWorldHint": true,
          "readOnlyHint": false,
          "title": "HTTP request"
        },
        "description": "Makes one HTTP(S) request with Go's HTTP client and returns status, headers and body. NOT read-only: any `method` (default GET) is sent as given, so POST, PUT or DELETE change the remote system. Follows redirects (up to 10); a non-2xx status is not an error, check `status_code`. The default timeout is 10 s (`timeout`, whole seconds) and the 30 s worker limit caps anything larger. The body is cut at `max_body` (default 1 MiB, max 10 MiB) and `truncated` is then true. `insecure` skips TLS verification. The daemon's proxy environment is honored unless the user has a `network:` policy in mcp-sudo.yaml; such a policy applies to every call and redirect hop, and a blocked destination fails to connect. Header and body values are redacted in the audit log. Returns JSON with `status_code`, `status`, `headers` (values comma-joined), `body`, `truncated`; `output_format` is ignored. For DNS use `network/nslookup`, for TCP reachability `network/ping`, for local files `files/read`.",
        "inputSchema": {
          "properties": {
            "body": {
              "description": "Request body",
              "type": "string"
            },
            "headers": {
              "additionalProperties": {
                "type": "string"
              },
              "description": "Request headers, e.g. {\"Content-Type\": \"application/json\"}",
              "type": "object"
            },
            "insecure": {
              "description": "Skip TLS certificate verification",
              "type": "boolean"
            },
            "max_body": {
              "description": "Return at most this many bytes of the response body (default 1048576 = 1 MiB, max 10 MiB); a cut body has truncated: true",
              "type": "integer"
            },
            "method": {
              "description": "HTTP method, default GET; any method is sent as given",
              "type": "string"
            },
            "timeout": {
              "description": "Whole seconds (default 10; the 30 s worker limit caps it)",
              "type": "number"
            },
            "url": {
              "description": "Full URL including scheme, e.g. https://example.com/api",
              "type": "string"
            }
          },
          "required": [
            "url"
          ],
          "type": "object"
        },
        "linuxctl_verb": "curl",
        "name": "network/curl",
        "tools_group": "network"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "ARP cache"
        },
        "description": "Shows the kernel's ARP cache (IPv4 address to MAC address) from /proc/net/arp in the daemon's network namespace. Read-only. It is a cache, not a scan: only hosts contacted recently appear, and IPv6 neighbours are not included. `interface` is an exact device name (`eth0`); omit it for all. For sockets and connections use `network/connections`, for reachability `network/ping`. Always returns JSON: an array of objects (ip_address, hw_type and flags as raw hex such as `0x1`, hw_address, mask, device). When nothing matches the output is `null`, not `[]`.",
        "inputSchema": {
          "properties": {
            "interface": {
              "description": "Exact interface name such as eth0; omit for all interfaces",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "arp",
        "name": "network/arp",
        "tools_group": "network"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": true,
          "readOnlyHint": true,
          "title": "TCP connect ping"
        },
        "description": "Tests TCP reachability: opens one TCP connection to `host:port` and closes it. This is NOT ICMP, so it needs a listening port (default 80) and says nothing about other ports or ICMP. Single attempt, no loss statistics; latency includes DNS resolution. Read-only, but it connects off-host and honors the user's `network:` policy in mcp-sudo.yaml. `timeout` is whole seconds (default 5). A failed connect is not a tool error: the JSON has `success: false` and an `error` text. Returns JSON with host, port, success, latency_ms and error. For DNS only use `network/nslookup`, for the hop path `network/trace-path`, for an HTTP check `network/curl`.",
        "inputSchema": {
          "properties": {
            "host": {
              "description": "Hostname or IP address to connect to",
              "type": "string"
            },
            "port": {
              "description": "TCP port, whole number (default 80)",
              "type": "number"
            },
            "timeout": {
              "description": "Seconds, whole number (default 5)",
              "type": "number"
            }
          },
          "required": [
            "host"
          ],
          "type": "object"
        },
        "linuxctl_verb": "ping",
        "name": "network/ping",
        "tools_group": "network"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "List sockets"
        },
        "description": "Lists TCP and UDP sockets in every state with their owning processes, like `ss -tuanp`, read natively from /proc/net (no ss needed). Read-only; unix and raw sockets are not included. `state` (case-insensitive, `_` and `-` interchangeable) filters by LISTEN (includes unconnected UDP), ESTABLISHED, TIME_WAIT, CLOSE_WAIT, SYN_SENT, ... or the groups connected/synchronized; an invalid state is an error listing the valid names. `port` matches the local OR peer port. The owning process (pid, fd) is shown only for sockets whose /proc/PID/fd you can read; other users' processes need `privileged: true` (needs a grant). Text output has `ss -tuanp` columns; `output_format: json` returns an array of objects (netid, state, recv_q, send_q, local_address, local_port, peer_address, peer_port, uid, inode, processes), `[]` when empty. For the ARP cache use `network/arp`, for interfaces and IPs the `network://interfaces` resource, for process details `processes/list`.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "port": {
              "description": "Match sockets whose local or peer port equals this",
              "type": "integer"
            },
            "privileged": {
              "description": "Run as root to see PIDs of other users. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused",
              "type": "boolean"
            },
            "state": {
              "description": "Filter by TCP state, case-insensitive (LISTEN/listening, ESTABLISHED, TIME_WAIT, CLOSE_WAIT, SYN_SENT, ...). Omit to list all sockets - active connections and listening ports.",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "connections",
        "name": "network/connections",
        "tools_group": "network"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Memory and swap usage"
        },
        "description": "Reports memory and swap use from /proc/meminfo. Read-only. `used` is total - free - (buffers + cached + reclaimable slab); `available` is the kernel's MemAvailable. Text output is a `free`-like table with Mem: and Swap: rows (bytes, or e.g. `1.8Gi` with `human_readable`); `detailed: true` returns the raw /proc/meminfo instead, but only for text output (it is ignored with `output_format: json`). JSON (also yaml/table/wide) returns an object (total, used, free, shared, buffCache, available, swap_total, swap_used, swap_free) in bytes; `human_readable` is ignored there. For per-process memory use `processes/top` or `processes/list`, for CPU load `cpu/load-average`.",
        "inputSchema": {
          "properties": {
            "detailed": {
              "description": "Return the raw /proc/meminfo instead of the summary (text output only; ignored with output_format json)",
              "type": "boolean"
            },
            "human_readable": {
              "description": "Sizes like free -h (1.8Gi); default is bytes",
              "type": "boolean"
            },
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "usage",
        "name": "memory/usage",
        "tools_group": "memory"
      },
      {
        "annotations": {
          "destructiveHint": true,
          "idempotentHint": false,
          "openWorldHint": false,
          "readOnlyHint": false,
          "title": "Control systemd service"
        },
        "description": "Starts, stops, restarts, reloads, enables or disables ONE systemd service over D-Bus (`.service` is appended when missing). Mutating. Ordinary users are usually refused by polkit, so `privileged: true` (needs a grant in mcp-sudo.yaml) is normally required. start, stop, restart and reload wait for the systemd job and return `Job N completed with status: done` (or failed, canceled, timeout, dependency, skipped); a slow one can hit the 30 s worker limit. `reload` asks the service to re-read its config without stopping it, only if the unit supports it. `enable` and `disable` only change whether it starts at boot; they do not start or stop it. To see state use `services/list` or the `service://<name>/status` resource, for logs `logs/journal-control`, for containers `docker/manage`, for a raw signal to a PID `processes/delete`.",
        "inputSchema": {
          "properties": {
            "action": {
              "description": "start, stop, restart and reload wait for the systemd job; enable and disable only change start at boot",
              "enum": [
                "start",
                "stop",
                "restart",
                "reload",
                "enable",
                "disable"
              ],
              "type": "string"
            },
            "privileged": {
              "description": "Run as root. Normally required (polkit); needs a grant for this tool in mcp-sudo.yaml",
              "type": "boolean"
            },
            "service": {
              "description": "Unit name; '.service' is appended if missing (e.g. 'kubelet' or 'kubelet.service')",
              "type": "string"
            }
          },
          "required": [
            "service",
            "action"
          ],
          "type": "object"
        },
        "linuxctl_verb": "services",
        "name": "services/manage",
        "tools_group": "system"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "List systemd services"
        },
        "description": "Lists systemd `.service` units that systemd currently has loaded, with load, active and sub state (D-Bus ListUnits). Read-only. An installed but never-loaded unit file may be missing, and timers, sockets and other unit types are not included. `pattern` supports only a leading and/or trailing `*` (`kube*`, `*ssh*`); without `*` it is an exact unit name including `.service`. The state filters are exact strings: `active_state` active/failed/inactive, `sub_state` running/exited/dead, `load_state` loaded/not-found. Text output is a block per unit plus a hint line, or `No services found matching the criteria.`; `output_format: json` returns an array of objects (name, description, load_state, active_state, sub_state), `[]` when empty. To change a service use `services/manage`, for one unit's details the `service://<name>/status` resource, for its logs `logs/journal-control`.",
        "inputSchema": {
          "properties": {
            "active_state": {
              "description": "Exact state: 'active', 'failed' or 'inactive'",
              "type": "string"
            },
            "load_state": {
              "description": "Exact state: 'loaded' or 'not-found'",
              "type": "string"
            },
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "pattern": {
              "description": "Only a leading and/or trailing * is supported ('kube*', '*ssh*'); without * an exact unit name including '.service'",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root (may be required depending on policies)",
              "type": "boolean"
            },
            "sub_state": {
              "description": "Exact state: 'running', 'exited' or 'dead'",
              "type": "string"
            }
          },
          "required": [],
          "type": "object"
        },
        "linuxctl_verb": "services",
        "name": "services/list",
        "tools_group": "system"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "List systemd timers"
        },
        "description": "Lists the systemd timers of the host - systemd's scheduler, the modern counterpart of cron - each with the unit it starts, its schedule (`OnCalendar=` and monotonic settings such as `OnBootSec=`), its next and last run, whether it is `Persistent` (catches up runs missed while the host was off) and its last result. Read-only. Use it to answer \"what runs on a schedule, and when next\": `services/list` shows only `.service` units, so timers never appear there. To inspect the unit a timer starts, use `linuxctl describe system <name>` or the `service://<name>/status` resource; a timer's own state is in this listing. Times are RFC 3339 UTC, and `never` means systemd reports none (a timer that has not fired yet, or one with no upcoming trigger). `pattern` accepts a leading and/or trailing `*` (`apt*`, `*.timer`, `*daily*`); the timer name includes `.timer`. Unprivileged calls need the host's systemd bus (a bare-metal or VM host); inside a container use `privileged: true`, which joins the host. `output_format: json` returns an array with the same fields; text is the default. Empty result means no timer matched. Managing timers (start, stop, enable) is not offered by this tool.",
        "inputSchema": {
          "properties": {
            "active_state": {
              "description": "Only timers in this active state (e.g. 'active', 'inactive', 'failed')",
              "type": "string"
            },
            "output_format": {
              "description": "'json' (also yaml/table/wide, which return the same JSON) for an array of objects; default is text",
              "type": "string"
            },
            "pattern": {
              "description": "Wildcard on the timer name (e.g. 'apt*', '*.timer', '*daily*'); no other wildcards",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root (needed inside a container to reach the host's systemd; needs a grant)",
              "type": "boolean"
            }
          },
          "required": [],
          "type": "object"
        },
        "linuxctl_verb": "get",
        "name": "timers/list",
        "tools_group": "timers"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Query systemd journal"
        },
        "description": "Reads the systemd journal (wraps `journalctl -n LINES --no-pager`). Read-only. Returns the last `lines` entries (default 100, the only size limit), oldest first unless `reverse`. Filter with `unit` (`sshd.service`), `since`/`until` in journalctl syntax (`1 hour ago`, `yesterday`, `2026-09-29 10:00`), `boot: true` (current boot) or `boot_offset` (-1 = previous boot; takes precedence over `boot`). Without root an ordinary user sees only their own entries unless in the `systemd-journal` or `adm` group. Requires privileged: true in containerized deployments (needs a grant), since journalctl only exists on the host, never in this daemon's own image. Plain text by default; `output_format: json` gives one JSON object per line (journalctl -o json), not an array. For kernel messages use `logs/dmesg`, for logins `logs/logins`, for container output `docker/logs`, for unit state `services/list`.",
        "inputSchema": {
          "properties": {
            "boot": {
              "description": "Restrict output to the current boot (journalctl -b)",
              "type": "boolean"
            },
            "boot_offset": {
              "description": "Select a prior boot relative to the current one, e.g. -1 for the previous boot (implies boot)",
              "type": "integer"
            },
            "lines": {
              "description": "Number of entries to tail (default 100; the only size limit)",
              "type": "integer"
            },
            "output_format": {
              "description": "json returns one JSON object per line (journalctl -o json); default is plain text",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root and join the host mount namespace - required in containerized deployments (needs a grant for this tool in mcp-sudo.yaml)",
              "type": "boolean"
            },
            "reverse": {
              "description": "Output newest entries first",
              "type": "boolean"
            },
            "since": {
              "description": "journalctl time syntax, e.g. '1 hour ago', 'today', '2026-09-29 10:00'",
              "type": "string"
            },
            "unit": {
              "description": "Filter by systemd unit (e.g., 'kubelet.service')",
              "type": "string"
            },
            "until": {
              "description": "Filter logs until a specific time (e.g., 'yesterday', '12:00')",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "journal",
        "name": "logs/journal-control",
        "tools_group": "logs"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Kernel ring buffer"
        },
        "description": "Reads the kernel ring buffer (wraps `dmesg --human`, relative timestamps). Read-only. Output is cut to the LAST 30 KiB with a `[WARNING: Output truncated to last 30KB]` prefix and cannot be paged; narrow it with `level`, a comma list of emerg, alert, crit, err, warn, notice, info, debug (`err,warn`). On hosts with `kernel.dmesg_restrict=1` an unprivileged call fails (dmesg's error is returned); use `privileged: true` (needs a grant). `output_format` is accepted and ignored: the reply is always plain text. For older history or service logs use `logs/journal-control`, for login records `logs/logins`, for drive faults `disks/health`.",
        "inputSchema": {
          "properties": {
            "level": {
              "description": "Comma-separated levels from emerg, alert, crit, err, warn, notice, info, debug (e.g. 'err,warn')",
              "type": "string"
            },
            "output_format": {
              "description": "Ignored - the reply is always plain text",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root - needed when kernel.dmesg_restrict=1. Needs a grant for this tool in mcp-sudo.yaml",
              "type": "boolean"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "dmesg",
        "name": "logs/dmesg",
        "tools_group": "logs"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Login history"
        },
        "description": "Lists login history (wraps `last`) or failed login attempts (`type: \"failed\"`, wraps `lastb`). Read-only; returns raw text, not JSON. `limit` keeps the N most recent entries and `user` filters by username; the trailing `wtmp begins...` summary line is dropped and an empty history returns an empty line. `last`/`lastb` must be installed on the host. For account details use `users/list`, for authentication messages `logs/journal-control`.",
        "inputSchema": {
          "properties": {
            "limit": {
              "description": "Only return this many most recent entries",
              "type": "integer"
            },
            "privileged": {
              "description": "Run as root - typically required for type \"failed\". Needs a grant for this tool in mcp-sudo.yaml",
              "type": "boolean"
            },
            "type": {
              "description": "\"success\" (default, wraps `last`) or \"failed\" (wraps `lastb`)",
              "enum": [
                "success",
                "failed"
              ],
              "type": "string"
            },
            "user": {
              "description": "Only return entries for this username",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "logins",
        "name": "logs/logins",
        "tools_group": "logs"
      },
      {
        "annotations": {
          "destructiveHint": true,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": false,
          "title": "Read/write sysctl"
        },
        "description": "Reads or writes a kernel parameter (sysctl) at runtime through /proc/sys. With `value` omitted it reads: `key` (dotted `net.ipv4.ip_forward` or slash form; a directory such as `net.ipv4` prints its subtree) returns `key = value` lines, and `read_all: true` (only when `key` is empty) prints every parameter, thousands of lines, uncapped. Reading needs no grant. With `value` it WRITES: that needs `privileged: true` and a grant, is refused for keys outside the user's `sysctl.write_keys` globs in mcp-sudo.yaml, and the reply shows the value the kernel now holds. Writes last until reboot; nothing is persisted to /etc/sysctl.d. For OS and kernel version use `system/os-release`, for memory figures `memory/usage`.",
        "inputSchema": {
          "properties": {
            "key": {
              "description": "Kernel parameter name, dotted (net.ipv4.ip_forward) or slash form (net/ipv4/conf/eth0.100/rp_filter). A directory (e.g. net.ipv4) reads its whole subtree.",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root - required for writes. Needs a grant for this tool in mcp-sudo.yaml",
              "type": "boolean"
            },
            "read_all": {
              "description": "Only when key is empty: read every parameter (thousands of lines, uncapped)",
              "type": "boolean"
            },
            "value": {
              "description": "Value to write (single line). If omitted, the parameter is read. Writing needs privileged: true and a matching sysctl.write_keys grant",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "sysctl",
        "name": "kernel/system-control",
        "tools_group": "kernel"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "CPU info"
        },
        "description": "Lists the machine's CPUs from /proc/cpuinfo. Read-only. Text output shows the number of logical processors and, for the FIRST processor only, vendor, model name, MHz (or BogoMIPS) and cache size. `output_format: json` (also yaml/table/wide) returns an array with one object per logical CPU using /proc/cpuinfo's own field names, which differ by architecture (x86 `model name`, `cpu MHz`, `flags`; ARM `CPU implementer`). It does not report sockets, cores or threads separately, and `topology_only` has no effect. For current load use `cpu/load-average`, for per-process CPU `processes/top`, for OS and kernel `system/os-release`.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "topology_only": {
              "description": "Has no effect: accepted but ignored, the output is the same",
              "type": "boolean"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "get",
        "name": "cpu/list",
        "tools_group": "cpu"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Load average"
        },
        "description": "Returns the 1, 5 and 15 minute load averages from sysinfo(2). Read-only; the only parameter is `output_format`. Load counts runnable plus uninterruptible tasks, not CPU percent: compare it with the number of logical CPUs from `cpu/list`. Text is `Load Average: 0.52, 0.48, 0.45`; `output_format: json` (also yaml/table/wide) returns an object with numbers `1_min`, `5_min`, `15_min`. For per-process CPU use `processes/top`, for memory `memory/usage`.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "load-average",
        "name": "cpu/load-average",
        "tools_group": "cpu"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "List block devices"
        },
        "description": "Lists block devices as a tree (like `lsblk`): disks, partitions, and LVM/dm-crypt/RAID volumes nested under the devices they are built on, with MAJ:MIN, RM, SIZE, RO, TYPE and MOUNTPOINTS, read from /sys/class/block (no lsblk needed). Read-only. Empty devices and RAM disks are hidden unless `all: true`; SIZE is in bytes unless `human_readable`. `output_format: json` (also yaml/table/wide) returns an object `blockdevices`, an array of objects (name, kname, maj:min, rm, size, size_bytes, ro, type, mountpoints) with nested `children`. In containerized deployments use `privileged: true` (needs a grant) to see the host's mount points. For free space or inodes use `disks/free`, for folder sizes `disks/usage`, for the mount table `disks/mounts`, for partition boundaries `disks/partitions`, for I/O counters `disks/performance`, for SMART `disks/health`.",
        "inputSchema": {
          "properties": {
            "all": {
              "description": "Include empty devices and RAM disks (lsblk -a)",
              "type": "boolean"
            },
            "human_readable": {
              "description": "SIZE like lsblk (60G); default is bytes (lsblk -b)",
              "type": "boolean"
            },
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root - in containerized deployments, reads the host's mount table for MOUNTPOINTS",
              "type": "boolean"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "get",
        "name": "disks/list",
        "tools_group": "disks"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "List mounts"
        },
        "description": "Lists mounted filesystems (device, mount point, type, options) from /proc/thread-self/mounts, sorted by mount point. Read-only. `fs_type` filters by exact type (`ext4`, `overlay`, `tmpfs`; no globs). It reads the daemon's own mount namespace, so in a container use `privileged: true` (needs a grant) to get the host's mounts. Text lines look like `/dev/sda1 on /mnt type ext4 (rw,...)`; `output_format: json` returns an array of objects (device, mount_point, fs_type, options), or `null` rather than `[]` when nothing matches (text is then empty). For block devices use `disks/list`, for the space used on a mount `disks/free`.",
        "inputSchema": {
          "properties": {
            "fs_type": {
              "description": "Exact filesystem type, no globs (e.g. 'ext4', 'overlay', 'tmpfs')",
              "type": "string"
            },
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused",
              "type": "boolean"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "mounts",
        "name": "disks/mounts",
        "tools_group": "disks"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Disk I/O counters"
        },
        "description": "Returns block-device I/O counters from /proc/diskstats: reads and writes completed and merged, sectors (512 bytes) and milliseconds spent, in-flight I/Os and weighted I/O time. Values are CUMULATIVE since boot, not rates and not iostat's per-interval figures; there is no %util or await, so sample twice and subtract to get a rate. Read-only. Without `device` all devices are listed except `loop*` and `ram*`; a name such as `sda` (find them with `disks/list`) selects one, and an unknown name gives `no such block device`. Text output is a table; `output_format: json` and `yaml` (real YAML here) return an array of objects with 14 fields (major, minor, device_name, reads_completed, ..., weighted_time_ios_ms). For capacity use `disks/free`, for SMART health `disks/health`.",
        "inputSchema": {
          "properties": {
            "device": {
              "description": "Block device name such as 'sda' (see disks/list); omit for all except loop* and ram*",
              "type": "string"
            },
            "output_format": {
              "description": "json or yaml (real YAML) for structured output; default is a text table",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "performance",
        "name": "disks/performance",
        "tools_group": "disks"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "SMART disk health"
        },
        "description": "Returns a drive's SMART data as smartctl's JSON (wraps `smartctl -j -a`; the smartmontools package must be installed or the call fails saying so). Read-only, but normally needs root: use `privileged: true` (needs a grant), since without it smartctl usually cannot open the device. `device` is a bare kernel name such as `sda` or `nvme0n1`, never a path; find names with `disks/list`. The reply is smartctl's JSON verbatim (keys such as smart_status, temperature, ata_smart_attributes or nvme_smart_health_information_log vary by drive type); a non-zero smartctl exit is not an error when it printed JSON, and virtual disks report SMART as unsupported. For I/O counters use `disks/performance`.",
        "inputSchema": {
          "properties": {
            "device": {
              "description": "Bare kernel device name such as 'sda' or 'nvme0n1' (never a path)",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root - required to read SMART data. Needs a grant for this tool in mcp-sudo.yaml",
              "type": "boolean"
            }
          },
          "required": [
            "device"
          ],
          "type": "object"
        },
        "linuxctl_verb": "health",
        "name": "disks/health",
        "tools_group": "disks"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Partition geometry"
        },
        "description": "Lists the partitions of a disk with start sector and size in sectors and bytes, read natively from /sys/class/block (no fdisk). Read-only. `device` names the PARENT DISK (`sda`, `nvme0n1`), not a partition; without it every disk's partitions are listed. A sector size of 512 bytes is assumed. It does not report partition type, label, UUID or filesystem: use `disks/list` or `disks/mounts` for those, and `disks/list` to find device names. A named disk without partitions is an error (`no partitions found for device`). Output is a text table; when `output_format` is set to any non-empty value (the parameter is accepted although not listed in the schema) it is an indented JSON array of objects (device, parent_disk, number, start_sector, size_sectors, size_bytes).",
        "inputSchema": {
          "properties": {
            "device": {
              "description": "Parent disk name such as 'sda' or 'nvme0n1' (not a partition); omit for all disks",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "partitions",
        "name": "disks/partitions",
        "tools_group": "disks"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": true,
          "readOnlyHint": true,
          "title": "Trace network path"
        },
        "description": "Traces the network path to a host by running the external `traceroute` binary (not tracepath; it must be installed on the host). Read-only, but it sends probe packets. `host` is a hostname or IP; `max_hops` defaults to traceroute's 30 and is capped at 255. There is no timeout parameter and the 30 s worker limit kills slow traces (30 hops x 3 probes can take minutes), so set a low `max_hops` such as 15. Non-responding hops show as `* * *`. Returns traceroute's raw text; there is no `output_format`. Use `network/ping` first for basic reachability, `network/nslookup` for DNS problems.",
        "inputSchema": {
          "properties": {
            "host": {
              "description": "Target hostname or IP address",
              "type": "string"
            },
            "max_hops": {
              "description": "Maximum hops (default 30, capped at 255); keep it low, the 30 s worker limit kills slow traces",
              "type": "integer"
            }
          },
          "required": [
            "host"
          ],
          "type": "object"
        },
        "linuxctl_verb": "trace-path",
        "name": "network/trace-path",
        "tools_group": "network"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "OS and kernel version"
        },
        "description": "Returns the Linux distribution and kernel version: the contents of /etc/os-release plus the uname line (system, host name, release, version, machine). Read-only. In a containerized daemon /etc/os-release is the container image's, not the host's. Text has an `OS Release Info:` block with the raw file and a `Kernel Info:` line; `output_format: json` (also yaml/table/wide) returns an object with `os_release` (the raw file text, not parsed into fields) and `kernel` (the uname line). For CPU details use `cpu/list`, for kernel parameters `kernel/system-control`.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "os-release",
        "name": "system/os-release",
        "tools_group": "system"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "List installed packages"
        },
        "description": "Lists installed packages by parsing the package database (dpkg on Debian/Ubuntu, apk on Alpine); rpm-based systems return an error, not supported yet. Read-only. The whole list is returned with no cap (hundreds of entries), so filter with `name`: an exact name or a glob (`openssh-*`, `*ssl*`). Only packages with status installed are listed. Text has a header `N packages installed (dpkg)` and a NAME VERSION ARCH table; `output_format: json` returns an array of objects (name, version, architecture), `[]` when none; there are no description or size fields. For OS and kernel version use `system/os-release`.",
        "inputSchema": {
          "properties": {
            "name": {
              "description": "Only packages whose name matches this glob or exact name (e.g. 'openssh-*', '*ssl*', 'curl')",
              "type": "string"
            },
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused",
              "type": "boolean"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "packages",
        "name": "system/packages",
        "tools_group": "system"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "List user accounts"
        },
        "description": "Lists local user accounts from /etc/passwd and /etc/group (uid, gid, home, shell, supplementary group memberships), sorted by uid. Read-only. Never reads /etc/shadow - this reports account identity, not credentials. Local files only: LDAP/SSSD users are not listed. `min_uid` (e.g. 1000) hides system accounts. Text lines look like `name (uid=N gid=N(group)) home=... shell=... groups=a,b`; `output_format: json` returns an array of objects (username, uid, gid, group_name, comment, home_dir, shell, groups). For who logged in use `logs/logins`, for your own root grants `auth/sudo-rules`.",
        "inputSchema": {
          "properties": {
            "min_uid": {
              "description": "Only include users with UID >= this value (e.g. 1000 to exclude system accounts)",
              "type": "integer"
            },
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root. Needs a grant for this tool in mcp-sudo.yaml, otherwise refused",
              "type": "boolean"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "get",
        "name": "users/list",
        "tools_group": "users"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "My root grants"
        },
        "description": "Shows which tools YOU may run as root: your `privileged` grants from mcp-sudo.yaml with their `paths`, `containers`, `prune`, `network` and `sysctl` restrictions. It does not list which tools you may call at all: unprivileged calls need no grant, except docker/* and daemon/reload-config, which always do. Read-only and answered by the daemon itself. Call it before a `privileged: true` request or after a permission-denied error. Text is `Your authorized privileged tools:` followed by JSON; `output_format: json` returns only that JSON, whose keys are capitalised Go field names (Tools, Resources, Allowed, Paths, Containers, Prune, Network, Sysctl). No grants gives `You have no privileged tools authorized in mcp-sudo.yaml.` (JSON: `{}`). After an operator edits grants, `daemon/reload-config` applies them.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "sudo-rules",
        "name": "auth/sudo-rules",
        "tools_group": "auth"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "List containers"
        },
        "description": "Lists Docker containers through the Engine API on the local socket (no `docker` CLI needed): name, image, state, status, ports. Read-only. Only running containers unless `all: true`; `state` (created, running, paused, exited, ...) filters to one state and implies `all`; `pattern` is a glob on the container name (`web-*`); `limit` returns the newest N. Every container is listed: the grant's `containers:` list applies only to docker/manage, docker/logs and docker/exec. Text is a block per container plus a hint line, or `No containers found matching the criteria.`; `output_format: json` (also yaml/table/wide) returns an array of objects (name, names, id, full_id, image, image_id, command, state, status, created, ports, labels), `[]` when empty. For logs use `docker/logs`, to start or stop `docker/manage`, for one container's details the `docker-container://<name>/status` resource. Always runs as root: no `privileged` argument, refused unless the user's grant has `allowed: true`.",
        "inputSchema": {
          "properties": {
            "all": {
              "description": "Include stopped containers (docker ps -a)",
              "type": "boolean"
            },
            "limit": {
              "description": "Return at most this many containers (newest first)",
              "type": "integer"
            },
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "pattern": {
              "description": "Only containers whose name matches this glob (e.g. 'web-*')",
              "type": "string"
            },
            "state": {
              "description": "Only containers in this state (implies all)",
              "enum": [
                "created",
                "restarting",
                "running",
                "removing",
                "paused",
                "exited",
                "dead"
              ],
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "get",
        "name": "docker/containers",
        "tools_group": "docker"
      },
      {
        "annotations": {
          "destructiveHint": true,
          "idempotentHint": false,
          "openWorldHint": false,
          "readOnlyHint": false,
          "title": "Container lifecycle"
        },
        "description": "Changes a container's lifecycle: start, stop, restart, kill, pause, unpause or remove; the container counterpart of `services/manage`. Mutating. Only containers in the grant's `containers:` list may be touched (name and ID are both checked; an empty list refuses everything). Always runs as root: no `privileged` argument, refused unless the grant has `allowed: true`. `stop` uses Docker's default 10 s grace. `remove` is irreversible for the container and deliberately fenced: it never sends `force` or volume removal, so a running container is refused (`stop or kill it first`) and anonymous volumes are kept. `start` on a running or `stop` on a stopped container reports `unchanged`, not an error. `container` is a name, full ID or ID prefix. Text: `Container NAME (ID12): stop succeeded.`; `output_format: json` returns container, id, action and result (`ok` or `unchanged`). For bulk cleanup use `docker/prune`, to run a command inside `docker/exec`, to check state first `docker/containers`.",
        "inputSchema": {
          "properties": {
            "action": {
              "description": "Action to perform",
              "enum": [
                "start",
                "stop",
                "restart",
                "kill",
                "pause",
                "unpause",
                "remove"
              ],
              "type": "string"
            },
            "container": {
              "description": "Container name, full ID or ID prefix",
              "type": "string"
            },
            "output_format": {
              "description": "json returns container, id, action, result; default is text",
              "type": "string"
            }
          },
          "required": [
            "container",
            "action"
          ],
          "type": "object"
        },
        "linuxctl_verb": "container",
        "name": "docker/manage",
        "tools_group": "docker"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "Container logs"
        },
        "description": "Reads one container's recent logs, the container counterpart of `logs/journal-control`: stdout and stderr interleaved in Docker's order, the last `lines` lines (default 100), never a follow. Read-only. `since`/`until` take a unix timestamp (`1759005000`) or an RFC3339 time; `stdout` and `stderr` default to true (both false returns nothing); `timestamps` prefixes each line. The answer is capped at 1 MiB and an oversized tail is cut at the END without a marker, so the newest lines can be lost: ask for fewer `lines`. Only containers in the grant's `containers:` list; always runs as root, refused unless the grant has `allowed: true`. Plain log text by default (`Container X (ID) has no log output for this selection.` when empty); `output_format: json` returns container, id, lines, logs. To run a command use `docker/exec`; for host logs `logs/journal-control`.",
        "inputSchema": {
          "properties": {
            "container": {
              "description": "Container name, full ID or ID prefix",
              "type": "string"
            },
            "lines": {
              "description": "Tail this many lines (default 100)",
              "type": "integer"
            },
            "output_format": {
              "description": "json returns container, id, lines, logs; default is raw log text",
              "type": "string"
            },
            "since": {
              "description": "Only entries after this time: unix timestamp (e.g. 1759005000) or RFC3339",
              "type": "string"
            },
            "stderr": {
              "description": "Include stderr (default true)",
              "type": "boolean"
            },
            "stdout": {
              "description": "Include stdout (default true)",
              "type": "boolean"
            },
            "timestamps": {
              "description": "Prefix every line with Docker's own timestamp",
              "type": "boolean"
            },
            "until": {
              "description": "Only entries before this time: unix timestamp or RFC3339",
              "type": "string"
            }
          },
          "required": [
            "container"
          ],
          "type": "object"
        },
        "linuxctl_verb": "logs",
        "name": "docker/logs",
        "tools_group": "docker"
      },
      {
        "annotations": {
          "destructiveHint": true,
          "idempotentHint": false,
          "openWorldHint": false,
          "readOnlyHint": false,
          "title": "Run command in container"
        },
        "description": "Runs ONE command inside a running container and returns its stdout, stderr and exit code: no TTY, no stdin, no follow. The highest-risk docker tool: arbitrary code, as the image's user (often root) unless `user` is set, with its own `containers:` grant list. Always runs as root at the Docker socket: no `privileged` argument, refused unless the grant has `allowed: true`. `command` is an argv array, not a shell line: `[\"sh\", \"-c\", \"ls /app\"]` for a shell. A non-zero exit code is not a tool error; read `exit_code`. `timeout` (seconds, default 30, capped at 300) is bounded by the worker limit, which is 30 s unless the operator sets `timeout_seconds` for docker/exec in daemon.yaml: then the answer is `timed out after N seconds` and the command keeps running in the container. Output is capped at 1 MiB. Text shows `Exit code: N` and the output; `output_format: json` returns container, id, command, exit_code, running, stdout, stderr. To read logs use `docker/logs`, to manage the container `docker/manage`.",
        "inputSchema": {
          "properties": {
            "command": {
              "description": "argv array, e.g. [\"sh\", \"-c\", \"ls /app\"]",
              "items": {
                "type": "string"
              },
              "type": "array"
            },
            "container": {
              "description": "Container name, full ID or ID prefix",
              "type": "string"
            },
            "output_format": {
              "description": "json returns container, id, command, exit_code, running, stdout, stderr; default is text",
              "type": "string"
            },
            "timeout": {
              "description": "Seconds to wait (default 30, values above 300 are capped at 300); the worker limit (30 s unless raised in daemon.yaml) ends the call first",
              "type": "integer"
            },
            "user": {
              "description": "Run as this user inside the container (name or uid[:gid]); default is the image's user",
              "type": "string"
            },
            "working_dir": {
              "description": "Working directory inside the container",
              "type": "string"
            }
          },
          "required": [
            "container",
            "command"
          ],
          "type": "object"
        },
        "linuxctl_verb": "exec",
        "name": "docker/exec",
        "tools_group": "docker"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "List images"
        },
        "description": "Lists Docker images through the Engine API on the local socket: tags, ID, size, creation time. Read-only (no pull, build or remove). Untagged intermediate layers are hidden unless `all: true`; untagged images that are shown appear as `<none>:<none>`. `pattern` is a glob on any tag or repository (`nginx*`, `*/api:*`). Text is tag lines with `ID | Size | Created`; `output_format: json` (also yaml/table/wide) returns an array of objects (id, full_id, repo_tags, repo_digests, size_bytes, created, containers = how many containers use it, labels), `[]` when empty. To delete unused images use `docker/prune` (target `images`, dangling only); for containers `docker/containers`; for one image's config the `docker-image://<ref>/inspect` resource. Always runs as root: no `privileged` argument, refused unless the grant has `allowed: true`.",
        "inputSchema": {
          "properties": {
            "all": {
              "description": "Include intermediate and untagged images (docker images -a)",
              "type": "boolean"
            },
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "pattern": {
              "description": "Only images whose tag or repository matches this glob (e.g. 'nginx*', '*/api:*')",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "images",
        "name": "docker/images",
        "tools_group": "docker"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "List volumes"
        },
        "description": "Lists Docker volumes with driver, mountpoint (a host path) and the names of the containers, running or stopped, that mount each. Read-only: volumes are never created or removed here. `pattern` is a glob on the volume name. Text is a block per volume; `output_format: json` (also yaml/table/wide) returns an array of objects (name, driver, mountpoint, created, scope, labels, options, in_use_by), `[]` when empty. `docker/prune` with target `volumes` removes anonymous unused volumes only. For one volume's details use the `docker-volume://<name>/inspect` resource, for containers `docker/containers`. Always runs as root: no `privileged` argument, refused unless the grant has `allowed: true`.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "pattern": {
              "description": "Only volumes whose name matches this glob",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "volumes",
        "name": "docker/volumes",
        "tools_group": "docker"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": true,
          "title": "List Docker networks"
        },
        "description": "Lists Docker networks (not the host's: for host interfaces and routes use the `network://` resources and `network/*` tools) with driver, scope, subnet, gateway and the containers attached with their addresses. Read-only. `pattern` is a glob on the name; `driver` an exact driver (bridge, host, none, overlay, macvlan). Text shows `Attached: (nothing)` for an empty network; `output_format: json` (also yaml/table/wide) returns an array of objects (name, id, full_id, driver, scope, created, internal, attachable, ingress, ipam_driver, subnets, gateways, containers, options, labels). Removing unused networks is `docker/prune` (target `networks`); one network's full detail is the `docker-network://<name>/inspect` resource. Always runs as root: no `privileged` argument, refused unless the grant has `allowed: true`.",
        "inputSchema": {
          "properties": {
            "driver": {
              "description": "Only networks using this driver (bridge, host, none, overlay, macvlan, ...)",
              "type": "string"
            },
            "output_format": {
              "description": "Use json for structured output (yaml, table and wide return the same JSON); default is text",
              "type": "string"
            },
            "pattern": {
              "description": "Only networks whose name matches this glob (e.g. 'app-*')",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "networks",
        "name": "docker/networks",
        "tools_group": "docker"
      },
      {
        "annotations": {
          "destructiveHint": true,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": false,
          "title": "Prune unused Docker objects"
        },
        "description": "Deletes ONE kind of unused Docker object per call: `target` is containers (stopped), images (dangling only), volumes (anonymous, unused only; named volumes are never touched), networks (unused) or build-cache. Mutating and irreversible, and it deletes objects you did not name; call again for the next kind (pruning containers is what makes images dangling). The user's grant needs a `prune:` list containing the target, otherwise the call is refused before anything is deleted. Always runs as root: no `privileged` argument, refused unless the grant has `allowed: true`. The 30 s worker limit can end the call while the Engine keeps pruning. Text is `TARGET: N removed, X MiB reclaimed` plus the removed IDs (networks report no size); `output_format: json` returns target, deleted, count, space_reclaimed_bytes. To remove one named container use `docker/manage` (`remove`); to see what exists first `docker/containers`, `docker/images`, `docker/volumes`, `docker/networks`.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "json returns target, deleted, count, space_reclaimed_bytes; default is text",
              "type": "string"
            },
            "target": {
              "description": "The one kind to reclaim: containers (stopped), images (dangling), volumes (anonymous, unused), networks (unused) or build-cache",
              "enum": [
                "containers",
                "images",
                "volumes",
                "networks",
                "build-cache"
              ],
              "type": "string"
            }
          },
          "required": [
            "target"
          ],
          "type": "object"
        },
        "linuxctl_verb": "prune",
        "name": "docker/prune",
        "tools_group": "docker"
      },
      {
        "annotations": {
          "destructiveHint": true,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": false,
          "title": "Read or replace a crontab"
        },
        "description": "Reads or replaces a user's crontab, the list of commands cron runs for that account on a schedule, and lists which accounts have one. Without `content` it reads: your own crontab as raw text exactly as `crontab -l` prints it (empty when you have none), or with `privileged: true` and no `user`, a table of the accounts that have a crontab and that your grant lets you view. With `content` it writes: the WHOLE crontab is replaced (an empty string clears it); the `crontab` command rejects a file it cannot parse and then the old crontab is unchanged; a missing final newline is added; the limit is 64 KiB. Your own crontab needs no grant and runs as you. Another account's needs `privileged: true` and a rule naming that account in your cron/manage grant (`users: {name: {view: true, edit: true}}`); `edit` implies `view`, root's crontab can be viewed but never edited, and cron.allow/cron.deny still apply to that account. WARNING: writing a crontab schedules commands as that user, and the job outlives this session and the revocation of your token; every write is audit-logged (size, lines, hash, never the content). To avoid overwriting a change made meanwhile, pass `if_match` with the sha256 you read (`output_format: json` returns it). For systemd timers use `timers/list`. Needs the `crontab` command on the host; inside a container a call sees the container's own crontabs.",
        "inputSchema": {
          "properties": {
            "content": {
              "description": "The complete new crontab; present = write (empty string clears it), absent = read. Standard crontab syntax: `m h dom mon dow command`, `@daily`, and NAME=value lines",
              "type": "string"
            },
            "if_match": {
              "description": "sha256 of the crontab as you read it; the write is refused if it has changed since",
              "type": "string"
            },
            "output_format": {
              "description": "'json' (also yaml/table/wide, which return the same JSON) returns {user, exists, lines, jobs, bytes, sha256, content} for a read; default is the raw text",
              "type": "string"
            },
            "privileged": {
              "description": "Needed to list crontabs or to act on another account (needs a cron/manage grant); ignored for your own crontab",
              "type": "boolean"
            },
            "user": {
              "description": "Account whose crontab to read or replace; default is your own. Another account needs privileged: true and a rule for it",
              "type": "string"
            }
          },
          "required": [],
          "type": "object"
        },
        "linuxctl_verb": "get",
        "name": "cron/manage",
        "tools_group": "crontabs"
      },
      {
        "annotations": {
          "destructiveHint": false,
          "idempotentHint": true,
          "openWorldHint": false,
          "readOnlyHint": false,
          "title": "Reload mcpd config"
        },
        "description": "Re-reads mcpd's config files (daemon.yaml, users.yaml, mcp-sudo.yaml) and applies them without a restart: users and tokens, per-user grants, rate limits and tool timeouts. The files themselves are edited on the host (linuxctl); this only reloads them. Mutating (replaces the in-memory config) and always needs `allowed: true` for `daemon/reload-config` in the caller's grant; there is no unprivileged mode. The files are validated strictly first (a misspelled key is an error): if any is invalid nothing changes and the error is returned. Sessions of removed users and of users whose token changed are closed, possibly the caller's own. Server settings (port, TLS, `worker.containerized`, Docker socket) still need a restart. Returns free text listing what changed. Takes no parameters; verify grants afterwards with `auth/sudo-rules`.",
        "inputSchema": {
          "properties": {},
          "type": "object"
        },
        "linuxctl_verb": "reload",
        "name": "daemon/reload-config",
        "tools_group": "daemon"
      }
    ]
  }
}
```

</details>

### What differs between users

Nothing about *which* tools are listed: every user sees all 48 tools, and a call the grant does not allow fails with an error that names what is missing (see [mcp-sudo.yaml](../configuration/mcp-sudo)). The only per-user difference is a note appended to the description of a few tools when the user holds a root grant for them - 6 of them (`disks/free`, `disks/mounts`, `files/list`, `logs/logins`, `system/packages`, `users/list`). Captured live for `disks/free`, for a user without and with the grant:

```json
{
  "name": "disks/free",
  "description": "... : true` (a grant, and a `paths:` entry for root) only for paths you cannot stat."
}
```

```json
{
  "name": "disks/free",
  "description": "... ry for root) only for paths you cannot stat. (Authorized for 'privileged: true')"
}
```

So an agent learns from `tools/list` alone where `privileged: true` would be honored - and cannot use the list to find out what other users were granted.

## `resources/list`

Fixed resources, read with `resources/read` by their `uri`; 11 of them. One entry:

```bash
# POST to the endpoint the SSE stream gave you (see the Overview), then read the reply on the stream
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from the SSE stream>" \
  -H "Authorization: Bearer $MCP_TOKEN" -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/list"}'
```

```json
{
  "description": "Native system uname information (kernel version, node name). Hint: For CPU hardware architecture use cpu/list tool.",
  "group": "system",
  "linuxctl_verb": "uname",
  "mimeType": "text/plain",
  "name": "OS Uname",
  "uri": "os://uname"
}
```

<details>
<summary><b>The whole response</b> (11 resources)</summary>

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "resources": [
      {
        "description": "Native system uname information (kernel version, node name). Hint: For CPU hardware architecture use cpu/list tool.",
        "group": "system",
        "linuxctl_verb": "uname",
        "mimeType": "text/plain",
        "name": "OS Uname",
        "uri": "os://uname"
      },
      {
        "description": "/etc/os-release information (distribution, version).",
        "group": "system",
        "linuxctl_verb": "release",
        "mimeType": "text/plain",
        "name": "OS Release",
        "uri": "os://release"
      },
      {
        "description": "Native system network hostname. Hint: To resolve IP addresses use network/nslookup tool.",
        "group": "system",
        "linuxctl_verb": "hostname",
        "mimeType": "text/plain",
        "name": "System Hostname",
        "uri": "system://hostname"
      },
      {
        "description": "Configured IANA timezone (e.g. America/New_York) plus current local offset and time.",
        "group": "system",
        "linuxctl_verb": "timezone",
        "mimeType": "text/plain",
        "name": "System Timezone",
        "uri": "system://timezone"
      },
      {
        "description": "Configured locale settings (LANG, LC_*).",
        "group": "system",
        "linuxctl_verb": "locale",
        "mimeType": "text/plain",
        "name": "System Locale",
        "uri": "system://locale"
      },
      {
        "description": "Network interfaces, assigned IP addresses, and detailed RX/TX traffic statistics for all interfaces.",
        "group": "network",
        "linuxctl_verb": "interfaces",
        "mimeType": "application/json",
        "name": "Network Interfaces",
        "uri": "network://interfaces"
      },
      {
        "description": "IPv4 Routing Table (/proc/net/route). Hint: Use network/ping to test reachability.",
        "group": "network",
        "linuxctl_verb": "routes",
        "mimeType": "application/json",
        "name": "Network Routes",
        "uri": "network://routes"
      },
      {
        "description": "Connected USB devices (lsusb equivalent). Lists vendors, products, and bus mapping.",
        "group": "devices",
        "linuxctl_verb": "usb",
        "mimeType": "application/json",
        "name": "USB Devices",
        "uri": "devices://usb"
      },
      {
        "description": "Connected PCI devices (lspci equivalent). Includes network cards, GPUs, and controllers.",
        "group": "devices",
        "linuxctl_verb": "pci",
        "mimeType": "application/json",
        "name": "PCI Devices",
        "uri": "devices://pci"
      },
      {
        "description": "Desktop Management Interface info (lshw/hwinfo equivalent). Detailed hardware specifications (RAM banks, BIOS, chassis).",
        "group": "devices",
        "linuxctl_verb": "dmi",
        "mimeType": "application/json",
        "name": "DMI Hardware Info",
        "uri": "devices://dmi"
      },
      {
        "description": "Loaded kernel drivers (lsmod equivalent). Hint: You can adjust kernel parameters via the kernel/system-control tool.",
        "group": "kernel",
        "linuxctl_verb": "modules",
        "mimeType": "application/json",
        "name": "Kernel Modules",
        "uri": "kernel://modules"
      }
    ]
  }
}
```

</details>

## `resources/templates/list`

Resources with a parameter in the URI (`{pid}`, `{name}`, a path), read with `resources/read` after filling it in; 11 of them. One entry:

```bash
# POST to the endpoint the SSE stream gave you (see the Overview), then read the reply on the stream
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from the SSE stream>" \
  -H "Authorization: Bearer $MCP_TOKEN" -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/templates/list"}'
```

```json
{
  "description": "Reads any file on the system. Append /stat for file metadata, /content for contents, or /type for file type.",
  "group": "files",
  "mimeType": "text/plain",
  "name": "File Reader",
  "uriTemplate": "file:///{path}"
}
```

<details>
<summary><b>The whole response</b> (11 templates)</summary>

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "resourceTemplates": [
      {
        "description": "Reads any file on the system. Append /stat for file metadata, /content for contents, or /type for file type.",
        "group": "files",
        "mimeType": "text/plain",
        "name": "File Reader",
        "uriTemplate": "file:///{path}"
      },
      {
        "description": "Hardware device metadata. Valid types: usb, pci, dmi. Useful for inspecting attached physical hardware.",
        "group": "devices",
        "mimeType": "application/json",
        "name": "Hardware Devices",
        "uriTemplate": "devices://{type}"
      },
      {
        "description": "Detailed properties and RX/TX traffic statistics of a specific network interface.",
        "group": "network",
        "linuxctl_verb": "interfaces",
        "mimeType": "application/json",
        "name": "Network Interface Detail",
        "uriTemplate": "network://interfaces/{name}"
      },
      {
        "description": "Exposes DBus service properties (ActiveState, LoadState, SubState). Best used alongside the services/manage tool to check if a service actually started.",
        "group": "system",
        "linuxctl_verb": "services",
        "mimeType": "application/json",
        "name": "Service Status",
        "uriTemplate": "service://{name}/status"
      },
      {
        "description": "Real-time I/O statistics for a specific block device (e.g. sda). Returns JSON.",
        "group": "disks",
        "mimeType": "application/json",
        "name": "Disk I/O Statistics",
        "uriTemplate": "disks://{name}/stats"
      },
      {
        "description": "One account's crontab, read-only. `{view}` is `text` (the crontab exactly as `crontab -l` prints it, empty when there is none) or `info` (JSON: exists, lines, jobs, bytes, sha256 - the hash to pass as `if_match` when replacing it with the cron/manage tool). Your own crontab needs no grant; another account's needs a `view` rule for it in your cron/manage grant. To change a crontab use the cron/manage tool; for systemd timers use timers/list.",
        "group": "crontabs",
        "linuxctl_verb": "crontab",
        "mimeType": "text/plain",
        "name": "User Crontab",
        "uriTemplate": "crontab://{user}/{view}"
      },
      {
        "description": "One Docker container's own view of itself, chosen with `{view}`: `status` is a computed summary (state, health, exit code, restart count, uptime, image, ports, limits); `inspect` is Docker's full raw config; `stats` is one CPU/memory/network/IO snapshot, not a stream; `top` is the processes running inside it. Hint: list containers with the docker/containers tool first.",
        "group": "docker",
        "linuxctl_verb": "container",
        "mimeType": "application/json",
        "name": "Docker Container Introspection",
        "uriTemplate": "docker-container://{name}/{view}"
      },
      {
        "description": "Full configuration of one Docker image (layers, env, entrypoint, labels, digests). The name may be a tag (nginx:alpine), a repository path (ghcr.io/org/api:v1) or an image ID. Hint: list images with the docker/images tool.",
        "group": "docker",
        "linuxctl_verb": "image",
        "mimeType": "application/json",
        "name": "Docker Image Inspect",
        "uriTemplate": "docker-image://{name}/inspect"
      },
      {
        "description": "Driver, mountpoint, options and labels of one Docker volume. Hint: list volumes - with the containers mounting each - using the docker/volumes tool.",
        "group": "docker",
        "linuxctl_verb": "volume",
        "mimeType": "application/json",
        "name": "Docker Volume Inspect",
        "uriTemplate": "docker-volume://{name}/inspect"
      },
      {
        "description": "Full configuration of one Docker network: driver, scope, IPAM (subnets, gateways, IP ranges), options, labels and every attached container's name, IPv4/IPv6 address and MAC. Read-only - nothing here connects, disconnects or removes. The scheme is deliberately prefixed: network:// is the host's own networking (network://interfaces, network://routes), so a Docker network cannot use the bare noun. Hint: list networks with the docker/networks tool.",
        "group": "docker",
        "linuxctl_verb": "network",
        "mimeType": "application/json",
        "name": "Docker Network Inspect",
        "uriTemplate": "docker-network://{name}/inspect"
      },
      {
        "description": "Reads process metadata from procfs. Valid targets: status, cmdline, environ, limits (rlimits, e.g. max open files), open_files (fd table - files, sockets, pipes). Hint: Find PIDs using the processes/list tool first.",
        "group": "processes",
        "mimeType": "application/json",
        "name": "Process Introspection",
        "uriTemplate": "process://{pid}/{target}"
      }
    ]
  }
}
```

</details>
