# manage

This package implements the `manage` tool/resource for the MCP daemon.

## Overview

Starts, stops, restarts, reloads, enables or disables ONE systemd service over D-Bus (`.service` is appended when missing). Mutating. Ordinary users are usually refused by polkit, so `privileged: true` (needs a grant in mcp-sudo.yaml) is normally required. start, stop, restart and reload wait for the systemd job and return `Job N completed with status: done` (or failed, canceled, timeout, dependency, skipped); a slow one can hit the 30 s worker limit. `reload` asks the service to re-read its config without stopping it, only if the unit supports it. `enable` and `disable` only change whether it starts at boot; they do not start or stop it. To see state use `services/list` or the `service://<name>/status` resource, for logs `logs/journal-control`, for containers `docker/manage`, for a raw signal to a PID `processes/delete`.

## Usage & Permissions

Refer to `configs/mcp-sudo.yaml` to see the default privilege requirements for this feature.
If this tool wraps a privileged binary, the worker execution will run as root if allowed by the configuration.
