# USB Devices

**URI**: `devices://usb`

Connected USB devices (lsusb equivalent). Lists vendors, products, and bus mapping.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource devices://usb
```

Output:
```json
[
  {
    "bus_id": "usb1",
    "vendor_id": "1d6b",
    "product_id": "0002",
    "manufacturer": "Linux 7.0.12-linuxkit vhci_hcd",
    "product": "USB/IP Virtual Host Controller"
  },
  {
    "bus_id": "usb2",
    "vendor_id": "1d6b",
    "product_id": "0003",
    "manufacturer": "Linux 7.0.12-linuxkit vhci_hcd",
    "product": "USB/IP Virtual Host Controller"
  }
]
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "devices://usb"}}'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "38",
  "result": {
    "contents": [
      {
        "mimeType": "application/json",
        "text": "[\n  {\n    \"bus_id\": \"usb1\",\n    \"vendor_id\": \"1d6b\",\n    \"product_id\": \"0002\",\n    \"manufacturer\": \"Linux 7.0.12-linuxkit vhci_hcd\",\n    \"product\": \"USB/IP Virtual Host Controller\"\n  },\n  {\n    \"bus_id\": \"usb2\",\n    \"vendor_id\": \"1d6b\",\n    \"product_id\": \"0003\",\n    \"manufacturer\": \"Linux 7.0.12-linuxkit vhci_hcd\",\n    \"product\": \"USB/IP Virtual Host Controller\"\n  }\n]",
        "uri": "devices://usb"
      }
    ]
  }
}
```

</details>

