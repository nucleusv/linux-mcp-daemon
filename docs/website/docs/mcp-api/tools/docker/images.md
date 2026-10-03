# Images

**Tool Name**: `docker/images`

Lists Docker images through the Engine API on the local socket: tags, ID, size, creation time. Read-only (no pull, build or remove). Untagged intermediate layers are hidden unless `all: true`; untagged images that are shown appear as `<none>:<none>`. `pattern` is a glob on any tag or repository (`nginx*`, `*/api:*`). Text is tag lines with `ID | Size | Created`; `output_format: json` (also yaml/table/wide) returns an array of objects (id, full_id, repo_tags, repo_digests, size_bytes, created, containers = how many containers use it, labels), `[]` when empty. To delete unused images use `docker/prune` (target `images`, dangling only); for containers `docker/containers`; for one image's config the `docker-image://<ref>/inspect` resource. Always runs as root: no `privileged` argument, refused unless the grant has `allowed: true`.

Sizes are Docker's own per-image size, which counts shared layers once per image - the numbers do not add up to the disk the images occupy together.

## Arguments

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| `all` | boolean | no | Include intermediate and untagged images (docker images -a) |
| `output_format` | string | no | Use json for structured output (yaml, table and wide return the same JSON); default is text |
| `pattern` | string | no | Only images whose tag or repository matches this glob (e.g. 'nginx*', '*/api:*') |

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get docker images
```

Output (Docker-in-Docker test host):
```text
nginx:alpine
  ID: df221db836e1 | Size: 89.5 MiB | Created: 2026-09-22 22:09:50 UTC

redis:alpine
  ID: 3811787313eb | Size: 152.5 MiB | Created: 2026-09-21 17:37:05 UTC

busybox:latest
  ID: fd7dc98638c8 | Size: 5.9 MiB | Created: 2026-05-13 02:21:49 UTC

Hint: For one image's layers, env and entrypoint, read docker-image://<name>/inspect
```

`-o json` adds the digests, the labels and the number of containers using the image:

```bash
linuxctl get docker images --pattern 'nginx*' -o json
```

Output:
```json
[
  {
    "containers": 1,
    "created": "2026-09-22T22:09:50Z",
    "full_id": "sha256:df221db836e1754089190208cee7eeda94f233197056426eda74a43ab1abeac2",
    "id": "df221db836e1",
    "labels": {
      "maintainer": "NGINX Docker Maintainers <docker-maint@nginx.com>"
    },
    "repo_digests": [
      "nginx@sha256:df221db836e1754089190208cee7eeda94f233197056426eda74a43ab1abeac2"
    ],
    "repo_tags": [
      "nginx:alpine"
    ],
    "size_bytes": 93853879
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "docker/images", "arguments": {"pattern": "nginx*", "output_format": "json"}}}'

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
        "text": "[\n  {\n    \"containers\": 1,\n    \"created\": \"2026-09-22T22:09:50Z\",\n    \"full_id\": \"sha256:df221db836e1754089190208cee7eeda94f233197056426eda74a43ab1abeac2\",\n    \"id\": \"df221db836e1\",\n    \"labels\": {\n      \"maintainer\": \"NGINX Docker Maintainers <docker-maint@nginx.com>\"\n    },\n    \"repo_digests\": [\n      \"nginx@sha256:df221db836e1754089190208cee7eeda94f233197056426eda74a43ab1abeac2\"\n    ],\n    \"repo_tags\": [\n      \"nginx:alpine\"\n    ],\n    \"size_bytes\": 93853879\n  }\n]"
      }
    ]
  }
}
```

</details>
