# dmesg

This package implements the `dmesg` tool/resource for the MCP daemon.

## Overview

Reads the kernel ring buffer (wraps `dmesg --human`, relative timestamps). Read-only. Output is cut to the LAST 30 KiB with a `[WARNING: Output truncated to last 30KB]` prefix and cannot be paged; narrow it with `level`, a comma list of emerg, alert, crit, err, warn, notice, info, debug (`err,warn`). On hosts with `kernel.dmesg_restrict=1` an unprivileged call fails (dmesg's error is returned); use `privileged: true` (needs a grant). `output_format` is accepted and ignored: the reply is always plain text. For older history or service logs use `logs/journal-control`, for login records `logs/logins`, for drive faults `disks/health`.

## Usage & Permissions

Refer to `configs/mcp-sudo.yaml` to see the default privilege requirements for this feature.
If this tool wraps a privileged binary, the worker execution will run as root if allowed by the configuration.
