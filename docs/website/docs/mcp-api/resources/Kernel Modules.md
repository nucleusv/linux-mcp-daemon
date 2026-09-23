# Kernel Modules

**URI**: `kernel://modules`

Loaded kernel drivers (lsmod equivalent). Hint: You can adjust kernel parameters via the kernel/system-control tool.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource kernel://modules
```

Output:
```json
[
  {
    "name": "selfowner",
    "size": "32768",
    "used_by_count": "-",
    "used_by": null,
    "state": "Live",
    "address": "0xffff80007a63c000"
  },
  {
    "name": "shiftfs",
    "size": "32768",
    "used_by_count": "-",
    "used_by": null,
    "state": "Live",
    "address": "0xffff80007a631000"
  },
  {
    "name": "rosetta",
    "size": "12288",
    "used_by_count": "-",
    "used_by": null,
    "state": "Live",
    "address": "0xffff80007a62b000"
  },
  {
    "name": "grpcfuse",
    "size": "12288",
    "used_by_count": "-",
    "used_by": null,
    "state": "Live",
    "address": "0xffff80007a625000"
  },
  {
    "name": "fakeowner",
    "size": "135168",
    "used_by_count": "-",
    "used_by": null,
    "state": "Live",
    "address": "0xffff80007a600000"
  }
]
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "kernel://modules"}}'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "40",
  "result": {
    "contents": [
      {
        "mimeType": "application/json",
        "text": "[\n  {\n    \"name\": \"selfowner\",\n    \"size\": \"32768\",\n    \"used_by_count\": \"-\",\n    \"used_by\": null,\n    \"state\": \"Live\",\n    \"address\": \"0xffff80007a63c000\"\n  },\n  {\n    \"name\": \"shiftfs\",\n    \"size\": \"32768\",\n    \"used_by_count\": \"-\",\n    \"used_by\": null,\n    \"state\": \"Live\",\n    \"address\": \"0xffff80007a631000\"\n  },\n  {\n    \"name\": \"rosetta\",\n    \"size\": \"12288\",\n    \"used_by_count\": \"-\",\n    \"used_by\": null,\n    \"state\": \"Live\",\n    \"address\": \"0xffff80007a62b000\"\n  },\n  {\n    \"name\": \"grpcfuse\",\n    \"size\": \"12288\",\n    \"used_by_count\": \"-\",\n    \"used_by\": null,\n    \"state\": \"Live\",\n    \"address\": \"0xffff80007a625000\"\n  },\n  {\n    \"name\": \"fakeowner\",\n    \"size\": \"135168\",\n    \"used_by_count\": \"-\",\n    \"used_by\": null,\n    \"state\": \"Live\",\n    \"address\": \"0xffff80007a600000\"\n  }\n]",
        "uri": "kernel://modules"
      }
    ]
  }
}
```

</details>

