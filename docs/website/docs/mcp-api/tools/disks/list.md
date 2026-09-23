# List

**Tool Name**: `disks/list`

Lists block devices as a tree (equivalent to `lsblk`): disks, their partitions, and LVM/dm-crypt/RAID volumes nested under the devices they're built on, with MAJ:MIN, RM, SIZE, RO, TYPE and MOUNTPOINTS. `json`/`yaml` output is the same tree under `blockdevices` (the shape of `lsblk -J`), with nested `children`. To check remaining free space or inode usage, use the disks/free tool. To check which folders are taking up the most space, use the disks/usage tool.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get disks
```

Output:
```text
NAME            MAJ:MIN RM  SIZE RO TYPE MOUNTPOINTS
sda               8:0    0   60G  0 disk
├─sda1            8:1    0    2G  0 part /boot
└─sda2            8:2    0   58G  0 part
  ├─vg4114-swap 252:0    0  3.7G  0 lvm  [SWAP]
  └─vg4114-root 252:1    0 54.3G  0 lvm  /
sr0              11:0    1    2K  1 rom
```

</details>

<details>
<summary><b>linuxctl (yaml tree)</b></summary>

```bash
linuxctl get disks -o yaml
```

Output (truncated):
```yaml
blockdevices:
    - name: sda
      kname: sda
      maj:min: "8:0"
      rm: false
      size: 60G
      size_bytes: 64424509440
      ro: false
      type: disk
      mountpoints: []
      children:
        - name: sda1
          kname: sda1
          maj:min: "8:1"
          ...
          mountpoints:
            - /boot
        - name: sda2
          ...
          children:
            - name: vg4114-swap
              kname: dm-0
              maj:min: "252:0"
              ...
              type: lvm
              mountpoints:
                - '[SWAP]'
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

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "7",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "NAME            MAJ:MIN RM  SIZE RO TYPE MOUNTPOINTS\nsda               8:0    0   60G  0 disk\n├─sda1            8:1    0    2G  0 part /boot\n└─sda2            8:2    0   58G  0 part\n  ├─vg4114-swap 252:0    0  3.7G  0 lvm  [SWAP]\n  └─vg4114-root 252:1    0 54.3G  0 lvm  /\nsr0              11:0    1    2K  1 rom\n"
      }
    ]
  }
}
```

</details>

