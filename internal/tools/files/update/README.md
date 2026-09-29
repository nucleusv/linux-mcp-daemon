# update

This package implements the `update` tool/resource for the MCP daemon.

## Overview

Edits part of an EXISTING file in place: appends text or replaces an inclusive line range. Mutating and not atomic; the file keeps its mode and owner. To write a whole file use `files/create`; to see the lines first use `files/read` with `start_line`/`end_line`. With `append: true` exactly `content` is added at the end (no newline is added, include your own) and the file is created if missing, though not its parent directories; append wins over any line range. Otherwise `start_line` AND `end_line` are both required (1-indexed, `end_line` >= `start_line`), the file must exist, and those lines are replaced by `content` (one trailing newline of `content` is ignored; an `end_line` past the end is clamped; a `start_line` past the end adds `content` as a new last line). There is no insert or delete mode: replacing with empty `content` leaves one empty line. Returns `Successfully appended to PATH` or `Successfully updated lines A-B in PATH`, no diff. `path` must be absolute; `privileged: true` (a grant, and for root a `paths:` entry) edits files you cannot write.

## Usage & Permissions

Refer to `configs/mcp-sudo.yaml` to see the default privilege requirements for this feature.
If this tool wraps a privileged binary, the worker execution will run as root if allowed by the configuration.
