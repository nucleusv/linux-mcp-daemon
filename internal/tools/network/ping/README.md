# network/ping

Measure TCP reachability and latency to a host.

This tool establishes a native TCP connection to a specified host and port (defaulting to port 80) and measures the exact round-trip connection latency in milliseconds.

## Input Schema

```json
{
  "type": "object",
  "properties": {
    "host": {
      "type": "string"
    },
    "port": {
      "type": "number",
      "description": "Defaults to 80"
    },
    "timeout": {
      "type": "number",
      "description": "Timeout in seconds (defaults to 5)"
    }
  },
  "required": ["host"]
}
```

## Destination restrictions
An optional per-user `network:` block on this tool's entry in `mcp-sudo.yaml` (`deny_private`, `allow`, `deny`) limits which addresses it may connect to - checked on the resolved IP at connect time. Unrestricted when absent. See `docs/website/docs/configuration/mcp-sudo.md` ("Restricting network destinations").
