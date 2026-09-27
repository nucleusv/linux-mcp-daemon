# docker/images

Lists images (`GET /images/json`) with tags, digests, size and how many containers use each. Read-only by construction: no pull, no build, no remove anywhere in this package.

Untagged intermediate layers are hidden unless `all: true`.

## Parameters
- `all` (boolean, optional): include intermediate and untagged images (`docker images -a`).
- `pattern` (string, optional): glob on tag or repository (`nginx*`, `*/api:*`).
- `output_format` (string, optional): `json`, `yaml`, `table`, `wide`; default text.

## Usage & Permissions
Root or nothing: needs `docker/images: {allowed: true}` in the caller's grant in `configs/mcp-sudo.yaml`. It names no container, so no `containers:` list applies.

`linuxctl get docker images` calls this tool. For one image's full config, read `image://<ref>/inspect`.
