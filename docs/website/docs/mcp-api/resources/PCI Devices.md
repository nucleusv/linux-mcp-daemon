# PCI Devices

**URI**: `devices://pci`

Connected PCI devices (lspci equivalent). Includes network cards, GPUs, and controllers.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource devices://pci
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "devices://pci"}}'
```

Real response (captured live):
```text
[
  {
    "slot": "0000:00:00.0",
    "vendor_id": "0x106b",
    "device_id": "0x1a05",
    "class": "0x060000"
  },
  {
    "slot": "0000:00:01.0",
    "vendor_id": "0x1af4",
    "device_id": "0x1041",
    "class": "0x020000"
  },
  {
    "slot": "0000:00:05.0",
    "vendor_id": "0x1af4",
    "device_id": "0x1043",
    "class": "0x078000"
  },
  {
    "slot": "0000:00:06.0",
    "vendor_id": "0x1af4",
    "device_id": "0x1042",
    "class": "0x018000"
  },
  {
    "slot": "0000:00:07.0",
    "vendor_id": "0x1af4",
    "device_id": "0x1042",
    "class": "0x018000"
  },
  {
    "slot": "0000:00:08.0",
    "vendor_id": "0x1af4",
    "device_id": "0x105a",
    "class": "0x018000"
  },
  {
    "slot": "0000:00:09.0",
    "vendor_id": "0x1af4",
    "device_id": "0x105a",
    "class": "0x018000"
  },
  {
    "slot": "0000:00:0a.0",
    "vendor_id": "0x1af4",
    "device_id": "0x105a",
    "cl
...
```

</details>
