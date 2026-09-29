# list

Lists the systemd timers of the host: the unit each one starts, its schedule (`OnCalendar=` and monotonic settings such as `OnBootSec=`), its next and last run, whether it is `Persistent`, and its last result. Read-only.

## Overview

Timers are systemd's scheduler, the modern counterpart of cron; on many hosts they are the only scheduler in use (the Ubuntu test VPS has 18 timers and no user crontab). `services/list` shows only `.service` units, so timers never appear there. This tool queries the same DBus API (`go-systemd/v22/dbus`, `ListUnitsContext`), keeps the `.timer` units, and reads each one's `Timer` properties (`GetUnitTypePropertiesContext`). No `systemctl` binary is invoked.

Results can be filtered by:
- `pattern`: wildcard on the timer name including `.timer` (a leading and/or trailing `*`; exact match otherwise)
- `active_state`: exact match (`active`, `inactive`, `failed`)

## Output

- Default: a text block per timer (state, next and last run, the unit it starts, the schedule, the description).
- `output_format: json|yaml|table|wide`: all return the same JSON array of `{name, description, load_state, active_state, sub_state, unit, next_run, last_run, schedule, persistent, result}`. Times are RFC 3339 UTC; `never` means systemd reports none (not fired yet, or no upcoming trigger).
- No match: `No timers found matching the criteria.` (text) or `[]` (JSON).

## Usage & Permissions

Unprivileged: no grant needed; it reads the system bus, which any user can. Inside a container the bus is not reachable, so use `privileged: true` there (needs a grant in `mcp-sudo.yaml`); a privileged worker joins the host namespace. Read-only; starting, stopping or enabling a timer is not offered by this tool.

To inspect the service a timer starts, use `linuxctl describe system <name>` or the `service://{name}/status` resource.
