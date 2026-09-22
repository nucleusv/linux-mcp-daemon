# list

**Tool Name**: `services/list`

Lists systemd services with optional filtering by name pattern, active state, load state, and sub state. Output includes `ActiveState`, `LoadState`, and `SubState` for each matching `.service` unit. To get detailed service properties and state for a single service, read the `service://{name}/status` resource. To control a service, use the `services/manage` tool.

## Arguments

| Argument | Type | Description |
| --- | --- | --- |
| `pattern` | string | Wildcard pattern to match service names (e.g., `kube*`, `*ssh*`) |
| `active_state` | string | Filter by active state (e.g., `active`, `failed`, `inactive`) |
| `load_state` | string | Filter by load state (e.g., `loaded`, `not-found`) |
| `sub_state` | string | Filter by sub state (e.g., `running`, `exited`, `dead`) |
| `output_format` | string | Desired output format (e.g. `json`, `table`, `wide`). Defaults to text |
| `privileged` | boolean | Run as root (may be required depending on policies) |
