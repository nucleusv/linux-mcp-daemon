# Nslookup

**Tool Name**: `nslookup`

Resolves a hostname to an IP address.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get network nslookup google.com
```

Output:
```text
{
  "host": "google.com",
  "records": [
    {
      "type": "CNAME",
      "value": "google.com."
    },
    {
      "type": "A",
      "value": "142.250.65.238"
    },
    {
      "type": "AAAA",
      "value": "2607:f8b0:4006:813::200e"
    },
    {
      "type": "TXT",
      "value": "v=spf1 include:_spf.google.com ~all"
    },
    {
      "type": "TXT",
      "value": "onetrust-domain-verification=0d477fe608074e6f9c12bca7826035cc"
    },
    {
      "type": "TXT",
      "value": "MS=E4A68B9AB2BB9670BCE15412F62916164C0B20BB"
    },
    {
      "type": "TXT",
      "value": "globalsign-smime-dv=CDYX+XFHUw2wml6/Gb8+59BsH31KzUr6c1l2BPvqKX8="
    },
    {
      "type": "TXT",
      "value": "Z29vZ2xl"
    },
    {
      "type": "TXT",
      "value": "apple-domain-verification=30afIBcvSuDV2PLX"
    },
    {
      "type": "TXT",
      "value": "facebook-domain-verification=22rm551cu4k0ab0b
...
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "network/nslookup", "arguments": {"host": "google.com"}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "13",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\n  \"host\": \"google.com\",\n  \"records\": [\n    {\n      \"type\": \"CNAME\",\n      \"value\": \"google.com.\"\n    },\n    {\n      \"type\": \"A\",\n      \"value\": \"142.250.65.238\"\n    },\n    {\n      \"type\": \"AAAA\",\n      \"value\": \"2607:f8b0:4006:813::200e\"\n    },\n    {\n      \"type\": \"TXT\",\n      \"value\": \"v=spf1 include:_spf.google.com ~all\"\n    },\n    {\n      \"type\": \"TXT\",\n      \"value\": \"onetrust-domain-verification=0d477fe608074e6f9c12bca7826035cc\"\n    },\n    {\n      \"type\": \"TXT\",\n      \"value\": \"MS=E4A68B9AB2BB9670BCE15412F62916164C0B20BB\"\n    },\n    {\n      \"type\": \"TXT\",\n      \"value\": \"globalsign-smime-dv=CDYX+XFHUw2wml6/Gb8+59BsH31KzUr6c1l2BPvqKX8=\"\n    },\n    {\n      \"type\": \"TXT\",\n      \"value\": \"Z29vZ2xl\"\n    },\n    {\n      \"type\": \"TXT\",\n      \"value\": \"apple-domain-verification=30afIBcvSuDV2PLX\"\n    },\n    {\n      \"type\": \"TXT\",\n      \"value\": \"facebook-domain-verification=22rm551cu4k0ab0b\n..."
      }
    ]
  }
}
```

</details>

