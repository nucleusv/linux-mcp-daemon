# cpu/get-cpu-info

Retrieves information about the system's CPU topology and architecture using `lscpu`.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `topology_only` | boolean | No | If true, strips out virtualization and cache details and only returns a basic table of CPU cores, sockets, and threads. |

## Examples

```json
{
  "topology_only": true
}
```
