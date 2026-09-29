# network/ping

Tests TCP reachability: opens one TCP connection to `host:port` and closes it. This is NOT ICMP, so it needs a listening port (default 80) and says nothing about other ports or ICMP. Single attempt, no loss statistics; latency includes DNS resolution. Read-only, but it connects off-host and honors the user's `network:` policy in mcp-sudo.yaml. `timeout` is whole seconds (default 5). A failed connect is not a tool error: the JSON has `success: false` and an `error` text. Returns JSON with host, port, success, latency_ms and error. For DNS only use `network/nslookup`, for the hop path `network/trace-path`, for an HTTP check `network/curl`.

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
