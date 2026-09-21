# network/list-connections

Lists active network connections and listening ports on the system using `ss -tulnp`.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `state` | string | No | Filter by TCP state (e.g., `LISTEN`, `ESTABLISHED`). |
| `port` | integer | No | Filter connections by a specific port number. |
| `privileged` | boolean | No | **Important**: Set to true to view process IDs (PIDs) owned by other users, including root. |

## Examples

```json
{
  "state": "LISTEN",
  "port": 80,
  "privileged": true
}
```
