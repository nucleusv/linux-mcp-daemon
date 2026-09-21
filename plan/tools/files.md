# Files Group Implementation Plan

This document tracks the standard Linux file utilities and their implementation status in the MCP Daemon.

## File Exploration and Status

- [x] **ls** (`files/list`): List directory contents.
- [ ] **lsof** (`files/get_open_files`): List open files.
- [ ] **stat**: Display file or file system status.
- [ ] **find**: Search for files in a directory hierarchy.
- [ ] **file**: Determine file type.
- [ ] **cat / head / tail / less / grep**: Output, stream, or search file contents.

## File Manipulation

- [ ] **touch / echo** (`files/create/file`): Create a new file or append to it.
- [ ] **vi / nano / sed** (`files/update/file`): Modify file contents.
