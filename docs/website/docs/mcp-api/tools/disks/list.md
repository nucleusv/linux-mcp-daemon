# List

**Tool Name**: `disks/list`

Lists block devices and partitions. To check remaining free space or inode usage, use the disks/free tool. To check which folders are taking up the most space, use the disks/usage tool.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get disks
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
# 1. Open the SSE stream (in the background) and capture the one-time POST endpoint
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

# 2. POST the tools/call request to that endpoint
curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "disks/list", "arguments": {}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Real response (captured live):
```text
NAME         SIZE(BYTES)  RO     RM    
nbd0         0            false  false 
nbd1         0            false  false 
nbd10        0            false  false 
nbd11        0            false  false 
nbd12        0            false  false 
nbd13        0            false  false 
nbd14        0            false  false 
nbd15        0            false  false 
nbd2         0            false  false 
nbd3         0            false  false 
nbd4         0            false  false 
nbd5         0            false  false 
nbd6         0            false  false 
nbd7         0            false  false 
nbd8         0            false  false 
nbd9         0            false  false 
ram0         4194304      false  false 
ram1         4194304      false  false 
ram10        4194304      false  false 
ram11        4194304      false  false 
ram12        4194304      false  false 
ram13        4194304
...
```

</details>
