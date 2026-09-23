# network/curl

Transfer data from a URL using a native Go HTTP client.

This tool acts as a native equivalent of `curl` or `wget`, returning the response status, headers, and body text without needing external binaries.

## Input Schema

```json
{
  "type": "object",
  "properties": {
    "url": {
      "type": "string"
    },
    "method": {
      "type": "string",
      "description": "HTTP method (e.g., GET, POST). Defaults to GET."
    },
    "body": {
      "type": "string",
      "description": "Optional request body."
    },
    "timeout": {
      "type": "number",
      "description": "Timeout in seconds. Defaults to 10."
    }
  },
  "required": ["url"]
}
```

## Destination restrictions
An optional per-user `network:` block on this tool's entry in `mcp-sudo.yaml` (`deny_private`, `allow`, `deny`) limits which addresses it may connect to - checked on the resolved IP at connect time, including every redirect hop. Unrestricted when absent. See `docs/website/docs/configuration/mcp-sudo.md` ("Restricting network destinations").
