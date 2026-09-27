# Networks

**Tool Name**: `docker/networks`

Lists Docker networks with driver, scope, subnet, gateway and the containers attached to each with their addresses (plus IPAM options and labels in `output_format: json`). Read-only by construction: nothing here creates or removes a network, or connects a container to one - removing an unused network is docker/prune's job. Note the scheme for a single network is `docker-network://`, not `network://`: that one is already the host's own networking (network://interfaces, network://routes). Hint: for one network's full IPAM, options and per-container MAC addresses read docker-network://\<name\>/inspect. This tool always runs as root (the Docker socket is root-owned), so it takes no `privileged` argument - being able to call it at all means it was granted in mcp-sudo.yaml.

`Attached` is computed from the container list, not from the network listing: `GET /networks` leaves its `Containers` map empty however many containers are on the network - only an inspect fills it. So the names here include stopped containers, which appear without an address (a stopped container keeps its network membership but not its IP). This is the same cross-reference `docker/volumes` makes for `In use by`.

Only the flags that are set are printed - `internal`, `attachable`, `ingress`, `ipv6`. `Internal: false` on every network of every host is noise; the JSON form has all four as booleans.

## Parameters

| Name | Type | Description |
|---|---|---|
| `pattern` | string | Only networks whose name matches this glob (`app*`). |
| `driver` | string | Only networks on this driver (`bridge`, `host`, `none`, `overlay`, `macvlan`, …), matched case-insensitively. |
| `output_format` | string | `json`, `yaml`, `table`, `wide`. Default is text. |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get docker networks
```

Output (Docker-in-Docker test host: `appnet` and `backend` are user-defined, `db-1` is on all three):
```text
none (c345b29bca19)
  Driver: null | Scope: local
  Attached: (nothing)

host (7e278d4782bd)
  Driver: host | Scope: local
  Attached: (nothing)

bridge (105e289e28b8)
  Driver: bridge | Scope: local
  Subnet: 172.18.0.0/16 | Gateway: 172.18.0.1
  Attached: db-1 (172.18.0.4), logger-1 (172.18.0.3), web-1 (172.18.0.2)

appnet (784013297bf8)
  Driver: bridge | Scope: local
  Subnet: 10.10.0.0/24 | Gateway: 10.10.0.1
  Attached: db-1 (10.10.0.3), web-1 (10.10.0.2)

backend (1341d9362901)
  Driver: bridge | Scope: local | internal, attachable
  Subnet: 10.20.0.0/24 | Gateway: 10.20.0.1
  Attached: db-1 (10.20.0.2)

Hint: For one network's IPAM, options and per-container addresses, read docker-network://<name>/inspect
```

```bash
linuxctl get docker networks --driver host
```

Output:
```text
host (7e278d4782bd)
  Driver: host | Scope: local
  Attached: (nothing)

Hint: For one network's IPAM, options and per-container addresses, read docker-network://<name>/inspect
```

```bash
linuxctl get docker networks --pattern appnet -o json
```

Output:
```json
[
  {
    "attachable": false,
    "containers": [
      "db-1 (10.10.0.3)",
      "web-1 (10.10.0.2)"
    ],
    "created": "2026-09-27T10:39:54.320776261Z",
    "driver": "bridge",
    "full_id": "784013297bf883171e64f96e7a43b58df73682cf5b16ffdb95504e6019eb3bdd",
    "gateways": [
      "10.10.0.1"
    ],
    "id": "784013297bf8",
    "ingress": false,
    "internal": false,
    "ipam_driver": "default",
    "ipv6": false,
    "labels": {
      "app": "demo",
      "owner": "fr013"
    },
    "name": "appnet",
    "options": {},
    "scope": "local",
    "subnets": [
      "10.10.0.0/24"
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "docker/networks", "arguments": {"pattern": "appnet"}}}'

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
                "text": "appnet (784013297bf8)\n  Driver: bridge | Scope: local\n  Subnet: 10.10.0.0/24 | Gateway: 10.10.0.1\n  Attached: db-1 (10.10.0.3), web-1 (10.10.0.2)\n\nHint: For one network's IPAM, options and per-container addresses, read docker-network://<name>/inspect\n"
            }
        ]
    }
}
```

</details>

## Permissions

Root or nothing, like every `docker/*` tool - the socket is `0660 root:docker`, and a uid outside that group cannot open it at all, so there is no unprivileged mode to fall back to. Grant it explicitly:

```yaml
docker/networks:
  allowed: true
```

No `containers:` list: this tool names no container, so one on the grant is a configuration error. Granting it at all lets an agent see every network on the host - its subnets, its gateways and every attached container's address.

Read-only is a property of the code, not of a flag: the package issues `GET /networks` and `GET /containers/json` and nothing else. See [permissions and risks](../../../configuration/permissions-and-risks) for why any Docker socket grant is close to host root regardless.
