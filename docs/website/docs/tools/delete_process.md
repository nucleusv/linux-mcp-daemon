# processes/kill-process

Terminates a specific process by its Process ID (PID). 

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `pid` | integer | Yes | The target PID to terminate. |
| `signal` | string | No | The signal to send. Defaults to `SIGTERM`. Examples: `SIGKILL`, `SIGINT`. |
| `privileged` | boolean | No | Set to true to run as root, allowing termination of processes owned by other users. |

## Examples

```json
{
  "pid": 1234,
  "signal": "SIGKILL",
  "privileged": true
}
```
