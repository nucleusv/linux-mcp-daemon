# disks/list-blocks

Lists block devices and their mount points using `lsblk`.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `all` | boolean | No | If true, includes empty devices (such as inactive loop devices or unmounted partitions) that are normally hidden. |

## Examples

```json
{
  "all": true
}
```
