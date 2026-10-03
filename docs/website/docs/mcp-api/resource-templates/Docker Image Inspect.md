# Docker Image Inspect

**URI Template**: `docker-image://{name}/inspect`

Full configuration of one Docker image (layers, env, entrypoint, labels, digests). The name may be a tag (nginx:alpine), a repository path (ghcr.io/org/api:v1) or an image ID. Hint: list images with the docker/images tool.

A read needs two grants in [mcp-sudo.yaml](../../configuration/mcp-sudo.md): the template itself (`docker-image://{name}/inspect`) and the internal worker it spawns (`docker/inspect`). There is no `containers:` list here - an image is not a container - so the grant is all or nothing. A registry path with slashes is fine; `..` in a reference is refused rather than cleaned, so a name can never address another Engine API endpoint.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource docker-image://nginx:alpine/inspect
```

Output (Docker-in-Docker test host, abridged - the full answer is 70 lines, adding every layer digest, `Metadata` and `Descriptor`):
```json
{
  "Id": "sha256:df221db836e1754089190208cee7eeda94f233197056426eda74a43ab1abeac2",
  "RepoTags": [
    "nginx:alpine"
  ],
  "RepoDigests": [
    "nginx@sha256:df221db836e1754089190208cee7eeda94f233197056426eda74a43ab1abeac2"
  ],
  "Comment": "buildkit.dockerfile.v0",
  "Created": "2026-09-22T22:09:50.079077168Z",
  "Config": {
    "ExposedPorts": {
      "80/tcp": {}
    },
    "Env": [
      "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
      "NGINX_VERSION=1.31.6",
      "PKG_RELEASE=1",
      "DYNPKG_RELEASE=1",
      "NJS_VERSION=1.0.1",
      "NJS_RELEASE=1",
      "ACME_VERSION=0.4.1"
    ],
    "Entrypoint": [
      "/docker-entrypoint.sh"
    ],
    "Cmd": [
      "nginx",
      "-g",
      "daemon off;"
    ],
    "WorkingDir": "/",
    "Labels": {
      "maintainer": "NGINX Docker Maintainers <docker-maint@nginx.com>"
    },
    "StopSignal": "SIGQUIT"
  },
  "Architecture": "arm64",
  "Variant": "v8",
  "Os": "linux",
  "Size": 92962093,
  "RootFS": {
    "Type": "layers",
    "Layers": [
      "sha256:1b349a3334531b575b01d409540dd8aca14ef29abddacd50f36643d7c949db4b",
      "sha256:f1c509e8c93b3f0f720ea882fa315edb1ec3afacc2151e0941838615925b11df"
    ]
  }
}
```

This is Docker's own JSON, passed through and indented - `Env`, `Entrypoint` and `Cmd` are what a container started from this image will run unless overridden, and `Architecture` is how to tell an arm64 image from an amd64 one without pulling it again.

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "docker-image://nginx:alpine/inspect"}}'
```

Response (abridged the same way):
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "contents": [
      {
        "mimeType": "application/json",
        "text": "{\n  \"Id\": \"sha256:df221db836e1754089190208cee7eeda94f233197056426eda74a43ab1abeac2\",\n  \"RepoTags\": [\n    \"nginx:alpine\"\n  ],\n  \"Created\": \"2026-09-22T22:09:50.079077168Z\",\n  \"Architecture\": \"arm64\",\n  \"Os\": \"linux\",\n  \"Size\": 92962093\n}",
        "uri": "docker-image://nginx:alpine/inspect"
      }
    ]
  }
}
```

</details>
