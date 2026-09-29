# network/curl

Makes one HTTP(S) request with Go's HTTP client and returns status, headers and body. NOT read-only: any `method` (default GET) is sent as given, so POST, PUT or DELETE change the remote system. Follows redirects (up to 10); a non-2xx status is not an error, check `status_code`. The default timeout is 10 s (`timeout`, whole seconds) and the 30 s worker limit caps anything larger. The body is cut at `max_body` (default 1 MiB, max 10 MiB) and `truncated` is then true. `insecure` skips TLS verification. The daemon's proxy environment is honored unless the user has a `network:` policy in mcp-sudo.yaml; such a policy applies to every call and redirect hop, and a blocked destination fails to connect. Header and body values are redacted in the audit log. Returns JSON with `status_code`, `status`, `headers` (values comma-joined), `body`, `truncated`; `output_format` is ignored. For DNS use `network/nslookup`, for TCP reachability `network/ping`, for local files `files/read`.

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
