# docker/volumes

Lists volumes (`GET /volumes`) with driver, mountpoint and **the containers currently mounting each**. Docker itself reports only a refcount (`UsageData.RefCount`), which answers "is this in use" but not "by what", so this package cross-references the container list and names them.

Read-only: no create, no remove. A volume is the one part of a Docker install holding data nothing else can rebuild.

## Parameters
- `pattern` (string, optional): glob on the volume name.
- `output_format` (string, optional): `json`, `yaml`, `table`, `wide`; default text.

## Usage & Permissions
Root or nothing: needs `docker/volumes: {allowed: true}` in the caller's grant in `configs/mcp-sudo.yaml`. It names no container, so no `containers:` list applies — but a mountpoint under `/var/lib/docker/volumes/…` is a real host path, readable with `files/read` by anyone granted that.

`linuxctl get docker volumes` calls this tool. For one volume's options and labels, read `volume://<name>/inspect`.
