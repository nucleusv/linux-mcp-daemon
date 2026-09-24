# Dmesg

**Tool Name**: `logs/dmesg`

Read the kernel ring buffer for hardware/driver logs.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get logs dmesg --privileged true
```

Output:
```text
[WARNING: Output truncated to last 30KB]
...
ort 1(veth0ef6e1f) entered disabled state
[  +0.000004] veth0ef6e1f: entered allmulticast mode
[  +0.000033] veth0ef6e1f: entered promiscuous mode
[  +0.103033] eth0: renamed from veth4d63741
[  +0.000641] docker0: port 1(veth0ef6e1f) entered blocking state
[  +0.000003] docker0: port 1(veth0ef6e1f) entered forwarding state
[  +0.088267] docker0: port 1(veth0ef6e1f) entered disabled state
[  +0.000039] veth4d63741: renamed from eth0
[  +0.010448] docker0: port 1(veth0ef6e1f) entered disabled state
[  +0.000722] veth0ef6e1f (unregistering): left allmulticast mode
[  +0.000002] veth0ef6e1f (unregistering): left promiscuous mode
[  +0.000002] docker0: port 1(veth0ef6e1f) entered disabled state
[Sep23 08:55] docker0: port 1(veth37722d0) entered blocking state
[  +0.000363] docker0: port 1(veth37722d0) entered disabled state
[  +0.000318] veth37722
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "logs/dmesg", "arguments": {"privileged": true}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "22",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "[WARNING: Output truncated to last 30KB]\n...\nort 1(veth0ef6e1f) entered disabled state\n[  +0.000004] veth0ef6e1f: entered allmulticast mode\n[  +0.000033] veth0ef6e1f: entered promiscuous mode\n[  +0.103033] eth0: renamed from veth4d63741\n[  +0.000641] docker0: port 1(veth0ef6e1f) entered blocking state\n[  +0.000003] docker0: port 1(veth0ef6e1f) entered forwarding state\n[  +0.088267] docker0: port 1(veth0ef6e1f) entered disabled state\n[  +0.000039] veth4d63741: renamed from eth0\n[  +0.010448] docker0: port 1(veth0ef6e1f) entered disabled state\n[  +0.000722] veth0ef6e1f (unregistering): left allmulticast mode\n[  +0.000002] veth0ef6e1f (unregistering): left promiscuous mode\n[  +0.000002] docker0: port 1(veth0ef6e1f) entered disabled state\n[Sep23 08:55] docker0: port 1(veth37722d0) entered blocking state\n[  +0.000363] docker0: port 1(veth37722d0) entered disabled state\n[  +0.000318] veth37722\n..."
      }
    ]
  }
}
```

</details>

