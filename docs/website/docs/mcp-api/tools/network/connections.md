# Connections

**Tool Name**: `network/connections`

Lists TCP and UDP sockets in every state with the processes that own them - what `ss -tuanp` shows - read natively from `/proc/net/{tcp,tcp6,udp,udp6}` and `/proc/<pid>/fd`, so it needs no `ss` on the host. For physical network links and IPs, use the `network://interfaces` resource.

| Argument | Meaning |
|---|---|
| `state` | `LISTEN`/`listening` (TCP listeners and unconnected UDP sockets), `ESTABLISHED`, `TIME_WAIT`, `CLOSE_WAIT`, `SYN_SENT`, ... in ss or kernel spelling, any case; or the groups `connected`, `synchronized`, `all`. Omitted: every socket. |
| `port` | only sockets whose local or peer port is this |
| `privileged` | run as root, so every socket's owning process is shown - otherwise only the calling user's own processes are |
| `output_format` | `json`, `yaml`, `table`, `wide`: one object per socket (below); otherwise ss-style text |

Two things only netlink knows, which `/proc/net` doesn't carry, differ from `ss`: a dual-stack IPv6 wildcard socket is shown with its real address `[::]:9091` (ss: `*:9091`), and a socket bound to a device has no `%iface` suffix (ss: `127.0.0.53%lo:53`).

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

Output captured live on an Ubuntu 24.04 host.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get network connections --state listening --privileged true
```

Output (Ubuntu 24.04 VPS):
```text
Netid  State   Recv-Q  Send-Q  Local Address:Port  Peer Address:Port  Process
tcp    LISTEN  0       0       0.0.0.0:22          0.0.0.0:*          users:(("systemd",pid=1,fd=118),("sshd",pid=837,fd=3))
tcp    LISTEN  0       0       127.0.0.1:45861     0.0.0.0:*          users:(("containerd",pid=813,fd=15))
tcp    LISTEN  0       0       [::]:22             [::]:*             users:(("systemd",pid=1,fd=119),("sshd",pid=837,fd=4))
tcp    LISTEN  0       0       [::]:9091           [::]:*             users:(("mcpd",pid=29372,fd=4))
tcp    LISTEN  0       0       [::]:9092           [::]:*             users:(("mcpd",pid=29447,fd=4))
udp    UNCONN  0       0       0.0.0.0:40414       0.0.0.0:*          users:(("docker-proxy",pid=7295,fd=7))
udp    UNCONN  0       0       203.0.113.117:68   0.0.0.0:*          users:(("systemd-network",pid=399,fd=21))
udp    UNCONN  0       0       [::]:40414          [::]:*             users:(("docker-proxy",pid=7300,fd=7))
```

Structured, for one port:

```bash
linuxctl get network connections --state listening --port 9091 --privileged true -o json
```

```json
[
  {
    "netid": "tcp",
    "state": "LISTEN",
    "recv_q": 0,
    "send_q": 0,
    "local_address": "::",
    "local_port": 9091,
    "peer_address": "::",
    "peer_port": 0,
    "uid": 0,
    "inode": 529827,
    "processes": [
      {
        "name": "mcpd",
        "pid": 124908,
        "fd": 4
      }
    ]
  }
]
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "network/connections", "arguments": {"state": "listening", "privileged": true}}}'

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
        "text": "Netid  State   Recv-Q  Send-Q  Local Address:Port  Peer Address:Port  Process\ntcp    LISTEN  0       0       0.0.0.0:22          0.0.0.0:*          users:((\"systemd\",pid=1,fd=118),(\"sshd\",pid=837,fd=3))\ntcp    LISTEN  0       0       127.0.0.1:45861     0.0.0.0:*          users:((\"containerd\",pid=813,fd=15))\ntcp    LISTEN  0       0       [::]:22             [::]:*             users:((\"systemd\",pid=1,fd=119),(\"sshd\",pid=837,fd=4))\ntcp    LISTEN  0       0       [::]:9091           [::]:*             users:((\"mcpd\",pid=29372,fd=4))\ntcp    LISTEN  0       0       [::]:9092           [::]:*             users:((\"mcpd\",pid=29447,fd=4))\nudp    UNCONN  0       0       0.0.0.0:40414       0.0.0.0:*          users:((\"docker-proxy\",pid=7295,fd=7))\nudp    UNCONN  0       0       203.0.113.117:68   0.0.0.0:*          users:((\"systemd-network\",pid=399,fd=21))\nudp    UNCONN  0       0       [::]:40414          [::]:*             users:((\"docker-proxy\",pid=7300,fd=7))\n"
      }
    ]
  }
}
```

</details>
