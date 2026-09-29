# create

This package implements the `create` tool/resource for the MCP daemon.

## Overview

Writes a whole file: creates it, or REPLACES the content of an existing one entirely. Mutating and not atomic. Missing parent directories are created (mode 0755); a new file gets mode 0644, an existing file keeps its mode. With `content` omitted or empty it only touches: creates an empty file or refreshes the modification time and never truncates an existing file. To change part of an existing file use `files/update` (append or replace lines); to check first whether a path exists use `files/list` or `files/find`; afterwards set mode or owner with `files/chmod`/`files/chown`. `content` is text and no trailing newline is added; `path` must be absolute. Writing where your account cannot needs `privileged: true` (a grant, and for root a `paths:` entry covering the path, which also refuses symlinks in the path; otherwise a symlink at the path is followed). Returns one line, `Successfully created and wrote to PATH` or `Successfully touched PATH`; failures are plain text such as `failed to write to file`.

## Usage & Permissions

Refer to `configs/mcp-sudo.yaml` to see the default privilege requirements for this feature.
If this tool wraps a privileged binary, the worker execution will run as root if allowed by the configuration.
