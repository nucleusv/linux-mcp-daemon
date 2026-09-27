# docker/exec

Runs one command inside a running container (`POST /containers/<id>/exec` + `/exec/<id>/start`) and returns its stdout, stderr and exit code. No TTY, no interactive attach, no follow: one command, one answer.

`command` is an argv array, not a shell line — `["sh", "-c", "ls /app"]` to use a shell. Output is capped at 1 MiB and the default timeout is 30s (300s maximum). A timed-out exec **keeps running inside the container**: the Engine API has no way to cancel one.

## Parameters
- `container` (string, required): name, full ID or ID prefix.
- `command` (array of string, required): argv.
- `user` (string, optional): name or `uid[:gid]`; default is the image's user.
- `working_dir` (string, optional).
- `timeout` (integer, optional): seconds, default 30, max 300.
- `output_format` (string, optional): `json`; default text.

## Usage & Permissions
⚠ The sharpest tool in the group — arbitrary code, as the image's user (usually root), inside whatever it targets. Needs `docker/exec: {allowed: true, containers: [...]}` in the caller's grant in `configs/mcp-sudo.yaml`, and that list is kept separate from every other docker tool's for exactly this reason. No `containers:` list refuses everything.

A container that mounts the host filesystem or runs privileged makes this host root by another route — see `docs/website/docs/configuration/permissions-and-risks.md`.

`linuxctl exec docker <name> sh -c "…"` calls this tool (the required array takes every remaining word as argv).
