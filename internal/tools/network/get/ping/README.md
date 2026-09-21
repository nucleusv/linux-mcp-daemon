# ping

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
