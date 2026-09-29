# docker/exec

Runs ONE command inside a running container and returns its stdout, stderr and exit code: no TTY, no stdin, no follow. The highest-risk docker tool: arbitrary code, as the image's user (often root) unless `user` is set, with its own `containers:` grant list. Always runs as root at the Docker socket: no `privileged` argument, refused unless the grant has `allowed: true`. `command` is an argv array, not a shell line: `["sh", "-c", "ls /app"]` for a shell. A non-zero exit code is not a tool error; read `exit_code`. `timeout` (seconds, default 30, capped at 300) is bounded by the worker limit, which is 30 s unless the operator sets `timeout_seconds` for docker/exec in daemon.yaml: then the answer is `timed out after N seconds` and the command keeps running in the container. Output is capped at 1 MiB. Text shows `Exit code: N` and the output; `output_format: json` returns container, id, command, exit_code, running, stdout, stderr. To read logs use `docker/logs`, to manage the container `docker/manage`.

`command` is an argv array, not a shell line — `["sh", "-c", "ls /app"]` to use a shell. Output is capped at 1 MiB. `timeout` defaults to 30s and is capped at 300s in the tool, but the worker limit (30s unless `tools: docker/exec: timeout_seconds` is raised in `daemon.yaml`) ends the call first. A timed-out exec **keeps running inside the container**: the Engine API has no way to cancel one.

## Parameters
- `container` (string, required): name, full ID or ID prefix.
- `command` (array of string, required): argv.
- `user` (string, optional): name or `uid[:gid]`; default is the image's user.
- `working_dir` (string, optional).
- `timeout` (integer, optional): seconds, default 30, capped at 300 by the tool and by the worker limit (see above).
- `output_format` (string, optional): `json`; default text.

## Usage & Permissions
⚠ The sharpest tool in the group — arbitrary code, as the image's user (usually root), inside whatever it targets. Needs `docker/exec: {allowed: true, containers: [...]}` in the caller's grant in `configs/mcp-sudo.yaml`, and that list is kept separate from every other docker tool's for exactly this reason. No `containers:` list refuses everything.

A container that mounts the host filesystem or runs privileged makes this host root by another route — see `docs/website/docs/configuration/permissions-and-risks.md`.

`linuxctl exec docker <name> sh -c "…"` calls this tool (the required array takes every remaining word as argv).
