# Docker Network Inspect

**URI Template**: `docker-network://{name}/inspect`

Full configuration of one Docker network: driver, scope, IPAM (subnets, gateways, IP ranges), options, labels and every attached container's name, IPv4/IPv6 address and MAC. Read-only - nothing here connects, disconnects or removes. Hint: list networks with the [docker/networks](../tools/docker/networks.md) tool.

The scheme is deliberately prefixed. `network://` is already the **host's own** networking - `network://interfaces`, `network://routes`, `network://interfaces/{name}` - so a Docker network cannot have the bare noun the way a container, image or volume does. That asymmetry with `container://`, `image://` and `volume://` is the whole reason for the `docker-` prefix; inside the docker group the keyword is still the plain noun (`linuxctl get docker network appnet`).

A read needs two grants in [mcp-sudo.yaml](../../configuration/mcp-sudo.md): the template itself (`docker-network://`) and the internal worker it spawns (`docker/inspect`). A network name is a single identifier, so a slash in it is refused rather than cleaned - the name can never address another Engine API endpoint.

Only an inspect fills `Containers`. The network **listing** (`GET /networks`, what `docker/networks` calls) returns that map empty however many containers are attached, which is why the tool cross-references the container list to print its `Attached` line. Here the addresses come with their prefix length (`10.10.0.2/24`) and each endpoint's MAC, neither of which the listing has.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource docker-network://appnet/inspect
```

Output (Docker-in-Docker test host, `web-1` and `db-1` attached):
```json
{
  "Name": "appnet",
  "Id": "784013297bf883171e64f96e7a43b58df73682cf5b16ffdb95504e6019eb3bdd",
  "Created": "2026-09-27T10:39:54.320776261Z",
  "Scope": "local",
  "Driver": "bridge",
  "EnableIPv4": true,
  "EnableIPv6": false,
  "IPAM": {
    "Driver": "default",
    "Options": {},
    "Config": [
      {
        "Subnet": "10.10.0.0/24",
        "Gateway": "10.10.0.1"
      }
    ]
  },
  "Internal": false,
  "Attachable": false,
  "Ingress": false,
  "ConfigFrom": {
    "Network": ""
  },
  "ConfigOnly": false,
  "Options": {},
  "Labels": {
    "app": "demo",
    "owner": "fr013"
  },
  "Containers": {
    "80d3cf9c8b3c4f5ed7b5a9d52c9f97d9759f585d0e9214f54646725eda7c0363": {
      "Name": "db-1",
      "EndpointID": "3dcfbfe6688fdaafc8e32dffb0587a51a3ff6d0a7ae46a19985abdf2ff631238",
      "MacAddress": "ea:b6:a8:d6:f2:a6",
      "IPv4Address": "10.10.0.3/24",
      "IPv6Address": ""
    },
    "e36aeba4369a725068b13142ef53c541abdfbaf857cfdc4b8370210e095a2099": {
      "Name": "web-1",
      "EndpointID": "5fe1fba9407d63987593f8f555baf418e6acd980dddc0679f381c1a2d45d296a",
      "MacAddress": "d2:f1:3d:54:42:8b",
      "IPv4Address": "10.10.0.2/24",
      "IPv6Address": ""
    }
  },
  "Status": {
    "IPAM": {
      "Subnets": {
        "10.10.0.0/24": {
          "IPsInUse": 5,
          "DynamicIPsAvailable": 251
        }
      }
    }
  }
}
```

`Status.IPAM` is the address-pool accounting - `IPsInUse: 5` on a /24 with `DynamicIPsAvailable: 251` is the gateway plus the two containers plus the reserved network and broadcast addresses. It is the fastest way to answer "is this network running out of addresses", which no other tool here reports.

`Containers` is keyed by full container ID, so joining it to a `docker/containers` listing is by `full_id`, not by name.

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "docker-network://appnet/inspect"}}'
```

Response (the `text` field holds the JSON above; shown here as it arrives on the wire):
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "contents": [
      {
        "mimeType": "application/json",
        "text": "{\n  \"Name\": \"appnet\",\n  \"Id\": \"784013297bf883171e64f96e7a43b58df73682cf5b16ffdb95504e6019eb3bdd\",\n  \"Created\": \"2026-09-27T10:39:54.320776261Z\",\n  \"Scope\": \"local\",\n  \"Driver\": \"bridge\",\n  \"EnableIPv4\": true,\n  \"EnableIPv6\": false,\n  \"IPAM\": {\n    \"Driver\": \"default\",\n    \"Options\": {},\n    \"Config\": [\n      {\n        \"Subnet\": \"10.10.0.0/24\",\n        \"Gateway\": \"10.10.0.1\"\n      }\n    ]\n  },\n  \"Internal\": false,\n  \"Attachable\": false,\n  \"Ingress\": false,\n  \"ConfigFrom\": {\n    \"Network\": \"\"\n  },\n  \"ConfigOnly\": false,\n  \"Options\": {},\n  \"Labels\": {\n    \"app\": \"demo\",\n    \"owner\": \"fr013\"\n  },\n  \"Containers\": {\n    \"80d3cf9c8b3c4f5ed7b5a9d52c9f97d9759f585d0e9214f54646725eda7c0363\": {\n      \"Name\": \"db-1\",\n      \"EndpointID\": \"3dcfbfe6688fdaafc8e32dffb0587a51a3ff6d0a7ae46a19985abdf2ff631238\",\n      \"MacAddress\": \"ea:b6:a8:d6:f2:a6\",\n      \"IPv4Address\": \"10.10.0.3/24\",\n      \"IPv6Address\": \"\"\n    },\n    \"e36aeba4369a725068b13142ef53c541abdfbaf857cfdc4b8370210e095a2099\": {\n      \"Name\": \"web-1\",\n      \"EndpointID\": \"5fe1fba9407d63987593f8f555baf418e6acd980dddc0679f381c1a2d45d296a\",\n      \"MacAddress\": \"d2:f1:3d:54:42:8b\",\n      \"IPv4Address\": \"10.10.0.2/24\",\n      \"IPv6Address\": \"\"\n    }\n  },\n  \"Status\": {\n    \"IPAM\": {\n      \"Subnets\": {\n        \"10.10.0.0/24\": {\n          \"IPsInUse\": 5,\n          \"DynamicIPsAvailable\": 251\n        }\n      }\n    }\n  }\n}\n",
        "uri": "docker-network://appnet/inspect"
      }
    ]
  }
}
```

</details>

## Permissions

Root or nothing, like everything that touches the Docker socket - it is `0660 root:docker`, so there is no unprivileged mode to fall back to. Without the grant the read is refused by name:

```text
Error: reading docker-network://appnet/inspect needs a "docker-network://" grant in this user's
resources: in mcp-sudo.yaml (the Docker socket is root-owned, so these reads have no
unprivileged mode) (Code: -32603)
```

Both grants:

```yaml
resources:
  docker-network://:
    - ""
tools:
  docker/inspect:      # the internal worker the template spawns
    allowed: true
```

See [permissions and risks](../../configuration/permissions-and-risks.md) for why any Docker socket grant is close to host root regardless.
