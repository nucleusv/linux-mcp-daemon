# list

This package implements the `services/list` tool for the MCP daemon.

## Overview

Lists systemd `.service` units by querying `go-systemd/v22/dbus` (`ListUnitsContext`), the same native DBus API used by `services/manage` and `services/status`. No external `systemctl` binary is invoked.

Results can be filtered by:
- `pattern`: simple wildcard match on unit name (`*` prefix/suffix/both supported; exact match otherwise)
- `active_state`, `load_state`, `sub_state`: exact match against the unit's DBus properties

## Output

- Default (no `output_format` or an unrecognized value): human-readable text block per service.
- `output_format: json|yaml|table|wide`: all currently return the same JSON array of `{name, description, load_state, active_state, sub_state}`. There is no distinct YAML/table/wide rendering yet — these values are accepted but not differentiated.

## Usage & Permissions

Refer to `configs/mcp-sudo.yaml` to see the default privilege requirements for this feature. The `privileged` argument is accepted but reading service state does not require root; it is present for symmetry with `services/manage` and to allow restricting access via `mcp-sudo.yaml` if needed.

To read detailed properties of a specific service, use the resource `service://{name}/status`.
