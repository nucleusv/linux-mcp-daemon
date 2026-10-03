# List

**Tool Name**: `disks/list`

Lists block devices as a tree (like `lsblk`): disks, partitions, and LVM/dm-crypt/RAID volumes nested under the devices they are built on, with MAJ:MIN, RM, SIZE, RO, TYPE and MOUNTPOINTS, read from /sys/class/block (no lsblk needed). Read-only. Empty devices and RAM disks are hidden unless `all: true`; SIZE is in bytes unless `human_readable`. `output_format: json` (also yaml/table/wide) returns an object `blockdevices`, an array of objects (name, kname, maj:min, rm, size, size_bytes, ro, type, mountpoints) with nested `children`. In containerized deployments use `privileged: true` (needs a grant) to see the host's mount points. For free space or inodes use `disks/free`, for folder sizes `disks/usage`, for the mount table `disks/mounts`, for partition boundaries `disks/partitions`, for I/O counters `disks/performance`, for SMART `disks/health`.

SIZE is in **bytes** by default; `human_readable: true` prints it like `lsblk`.

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `all` | boolean | no | Include empty devices and RAM disks (lsblk -a) |
| `human_readable` | boolean | no | SIZE like lsblk (60G); default is bytes (lsblk -b) |
| `output_format` | string | no | Use json for structured output (yaml, table and wide return the same JSON); default is text |
| `privileged` | boolean | no | Run as root - in containerized deployments, reads the host's mount table for MOUNTPOINTS |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get disks
```

Output (Ubuntu 24.04 VPS, 2 GB RAM):
```text
NAME            MAJ:MIN RM        SIZE RO TYPE MOUNTPOINTS
sda               8:0    0 64424509440  0 disk
├─sda1            8:1    0  2146435072  0 part /boot
└─sda2            8:2    0 62277025792  0 part
  ├─vg4114-swap 252:0    0  3955228672  0 lvm  [SWAP]
  └─vg4114-root 252:1    0 58317602816  0 lvm  /
sr0              11:0    1        2048  1 rom
```

</details>

<details>
<summary><b>linuxctl (human_readable)</b></summary>

```bash
linuxctl get disks --human_readable true
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

Output:
```yaml
blockdevices:
    - name: sda
      kname: sda
      maj:min: "8:0"
      rm: false
      size: "64424509440"
      size_bytes: 64424509440
      ro: false
      type: disk
      mountpoints: []
      children:
        - name: sda1
          kname: sda1
          maj:min: "8:1"
          rm: false
          size: "2146435072"
          size_bytes: 2146435072
          ro: false
          type: part
          mountpoints:
            - /boot
        - name: sda2
          kname: sda2
          maj:min: "8:2"
          rm: false
          size: "62277025792"
          size_bytes: 62277025792
          ro: false
          type: part
          mountpoints: []
          children:
            - name: vg4114-swap
              kname: dm-0
              maj:min: "252:0"
              rm: false
              size: "3955228672"
              size_bytes: 3955228672
              ro: false
              type: lvm
              mountpoints:
                - '[SWAP]'
            - name: vg4114-root
              kname: dm-1
              maj:min: "252:1"
              rm: false
              size: "58317602816"
              size_bytes: 58317602816
              ro: false
              type: lvm
              mountpoints:
                - /
    - name: sr0
      kname: sr0
      maj:min: "11:0"
      rm: true
      size: "2048"
      size_bytes: 2048
      ro: true
      type: rom
      mountpoints: []
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "disks/list", "arguments": {}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "NAME            MAJ:MIN RM        SIZE RO TYPE MOUNTPOINTS\nsda               8:0    0 64424509440  0 disk\n├─sda1            8:1    0  2146435072  0 part /boot\n└─sda2            8:2    0 62277025792  0 part\n  ├─vg4114-swap 252:0    0  3955228672  0 lvm  [SWAP]\n  └─vg4114-root 252:1    0 58317602816  0 lvm  /\nsr0              11:0    1        2048  1 rom\n"
      }
    ]
  }
}
```

</details>
