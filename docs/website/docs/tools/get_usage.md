# memory/get-memory

Retrieves memory and swap utilization information.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `detailed` | boolean | No | If false, returns human-readable summaries (`free -h`). If true, returns the complete, raw contents of `/proc/meminfo`. |

## Examples

```json
{
  "detailed": false
}
```
