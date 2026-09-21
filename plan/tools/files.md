# Files Group Implementation Plan

This document tracks the standard Linux file utilities and their implementation status in the MCP Daemon.

## File Exploration and Status (Tools)

- [x] **ls** (`files/list`): List directory contents.
- [ ] **find**: Search for files in a directory hierarchy.
- [ ] **lsof** (`files/open`): List open files.

## File Metadata and Contents (Resource Templates)
*Because reading file content and metadata is inherently read-only, we should expose these using MCP Resource Templates rather than tools.*

- [ ] **file://{path}/stat**: Expose file metadata (size, permissions, owner, modified time, etc.) via the `stat` utility natively.
- [ ] **file://{path}/content**: Expose the actual file contents (like `cat`, `head`, `tail`). **Crucially, to avoid blowing up the AI context window with 20MB files, this resource MUST strictly truncate output (e.g. max 100KB) and append a warning if the file was truncated.**
- [ ] **file://{path}/type**: Expose the file type using the `file` utility.

## File Manipulation and Precision Reading (Tools)

- [ ] **read** (`files/read`): A tool specifically designed for precision reading of large files. It MUST accept `offset`/`limit` (bytes) or `start_line`/`end_line` arguments to stream or chunk large files safely into context.
- [ ] **touch / echo** (`files/create`): Create a new file or append to it.
- [ ] **vi / nano / sed** (`files/update`): Modify file contents.
