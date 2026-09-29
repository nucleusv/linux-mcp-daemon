# files/read

Reads a text file and returns its contents as raw text (no line numbers or metadata). Read-only; `path` must be absolute. To find a file use `files/find`, for size or mode `files/list`, to check whether it is binary `files/filetype` (binary files are refused with `cannot read binary file`). Selection: `start_line`/`end_line` (1-indexed, inclusive; `start_line` alone reads to the end, `end_line` alone starts at line 1) take precedence over the byte range `offset`/`limit`. With no selection, or `offset` without `limit`, at most 10240 bytes come back followed by a `[WARNING: File truncated ...]` line, so page large files with `start_line`/`end_line` or pass `limit`. There is no streaming: one call reads into memory, and a line over 64 KiB fails line mode. A `start_line` past the end is an error; an empty file returns an empty string. `privileged: true` (needs a grant, and a `paths:` entry for root) reads files your account cannot.


## Input Schema

```json
{
  "type": "object",
  "properties": {
    "path": {"type": "string", "description": "Absolute path of the file to read."},
    "start_line": {"type": "integer"}, "end_line": {"type": "integer"},
    "offset": {"type": "integer"}, "limit": {"type": "integer"},
    "privileged": {"type": "boolean"}
  },
  "required": ["path"]
}
```

Also serves the `file:///{path}` resource. See the tool description above for the line/byte precedence and the 10240-byte default cap.
