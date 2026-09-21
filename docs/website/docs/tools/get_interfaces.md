# network/list-interfaces

Returns the configuration and status of all network interfaces on the system using `ip address show`.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `up_only` | boolean | No | Only show interfaces whose state is UP. |
| `privileged` | boolean | No | Run as root (usually not required for viewing IP addresses). |

## Examples

```json
{
  "up_only": true
}
```
