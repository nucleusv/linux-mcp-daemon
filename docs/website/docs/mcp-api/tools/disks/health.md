# Health

**Tool Name**: `disks/health`

Retrieves detailed SMART health data for a drive (equivalent to smartctl -j -a). Returns JSON containing self-assessment test results, temperature, wear leveling, and sector errors. Must be run as root (privileged: true). Use this to diagnose failing hardware.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get disks health vda
```

Output:
```json
{
  "json_format_version": [
    1,
    0
  ],
  "smartctl": {
    "version": [
      7,
      4
    ],
    "pre_release": false,
    "svn_revision": "5530",
    "platform_info": "aarch64-linux-7.0.12-linuxkit",
    "build_info": "(local build)",
    "argv": [
      "smartctl",
      "-j",
      "-a",
      "/dev/vda"
    ],
    "messages": [
      {
        "string": "/dev/vda: Unable to detect device type",
        "severity": "error"
      }
    ],
    "exit_status": 1
  },
  "local_time": {
    "time_t": 1790163825,
    "asctime": "Wed Sep 23 11:43:45 2026 UTC"
  }
}
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
# 1. Open the SSE stream (in the background) and capture the one-time POST endpoint
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

# 2. POST the tools/call request to that endpoint
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "disks/health", "arguments": {"device": "vda"}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "10",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\n  \"json_format_version\": [\n    1,\n    0\n  ],\n  \"smartctl\": {\n    \"version\": [\n      7,\n      4\n    ],\n    \"pre_release\": false,\n    \"svn_revision\": \"5530\",\n    \"platform_info\": \"aarch64-linux-7.0.12-linuxkit\",\n    \"build_info\": \"(local build)\",\n    \"argv\": [\n      \"smartctl\",\n      \"-j\",\n      \"-a\",\n      \"/dev/vda\"\n    ],\n    \"messages\": [\n      {\n        \"string\": \"/dev/vda: Unable to detect device type\",\n        \"severity\": \"error\"\n      }\n    ],\n    \"exit_status\": 1\n  },\n  \"local_time\": {\n    \"time_t\": 1790163825,\n    \"asctime\": \"Wed Sep 23 11:43:45 2026 UTC\"\n  }\n}"
      }
    ]
  }
}
```

</details>

