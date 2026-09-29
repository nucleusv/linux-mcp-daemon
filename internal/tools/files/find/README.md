# find

This package implements the `find` tool/resource for the MCP daemon.

## Overview

Searches a directory tree (default `/`) by name, type, age or size, like `find`. Read-only; never follows symlinks; skips `/proc`, `/sys`, `/dev` and `/run` when the search starts at `/` or above them (not when `path` is inside one). Use `files/list` to see one known directory and `disks/usage` to see which folders take the space. All filters are ANDed and none is required. There is NO result cap: `path: /` without a filter lists every file on the host, so give `name`, `type` or `max_depth`; the 30 s worker timeout applies. Syntax: `name` is a case-sensitive glob on the base name only (`*.log`, not a path); `type` is one of `f d l b c p s` or a comma list (`f,d`); `mtime` in days: `+7` older than 7 days, `-1` within the last day, `7` exactly 7 days; `size`: `+100M` larger, `-10k` smaller (units b c w k M G, rounded up); `max_depth` 1 = direct children, omitted or 0 = unlimited. Unreadable directories are skipped silently. Text output: one `SIZE PATH` line per match, sorted by path (no size for directories; empty output = no match). `output_format: json` returns an array of objects (path, size in bytes, type).

## Usage & Permissions

Refer to `configs/mcp-sudo.yaml` to see the default privilege requirements for this feature.
If this tool wraps a privileged binary, the worker execution will run as root if allowed by the configuration.
