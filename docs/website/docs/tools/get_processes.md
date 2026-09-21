# processes/list-processes

Lists currently running processes on the host system. By default, it returns all processes running in the namespace of the Daemon, or host-wide if running fully privileged on the host.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `user` | string | No | Filter processes by a specific username. |
| `sort_by` | string | No | Sort results by `cpu`, `mem`, or `pid`. |
| `limit` | integer | No | Truncate the output to a specific number of lines. |
| `privileged` | boolean | No | Run as root to see isolated or other users' processes. |

## Examples

```json
{
  "sort_by": "cpu",
  "limit": 10,
  "privileged": true
}
```
