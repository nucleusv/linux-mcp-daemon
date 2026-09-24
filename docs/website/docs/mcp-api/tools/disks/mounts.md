# Mounts

**Tool Name**: `disks/mounts`

Lists mounted filesystems (device, mount point, type, options) - equivalent to `mount`/`findmnt`'s basic view. Use disks/list for block devices instead.

When `mcpd` runs containerized, passing `privileged: true` automatically shows the real host's mount table instead of the daemon's own container's - see [Master Daemon Configuration](../../../configuration/daemon.md)'s `worker.containerized` setting.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get disks mounts
```

Output:
```text
overlay on / type overlay (rw,relatime,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1317/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1316/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1315/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1314/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1272/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1271/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1224/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/265/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/264/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1318/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshot
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "disks/mounts", "arguments": {}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "8",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "overlay on / type overlay (rw,relatime,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1317/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1316/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1315/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1314/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1272/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1271/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1224/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/265/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/264/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1318/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshot\n..."
      }
    ]
  }
}
```

</details>

