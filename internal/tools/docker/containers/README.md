# docker/containers

Lists containers from the Engine API over `/var/run/docker.sock` (`GET /containers/json`), parsed with the stdlib — the `docker` CLI is never invoked and need not be installed on the host.

Running containers only by default; `all: true` includes exited/created ones, and `state` filters to one state (implying `all`).

## Parameters
- `all` (boolean, optional): include stopped containers (`docker ps -a`).
- `pattern` (string, optional): glob on the container name (`web-*`).
- `state` (string, optional): `created`, `restarting`, `running`, `removing`, `paused`, `exited`, `dead`.
- `limit` (integer, optional): return at most this many.
- `output_format` (string, optional): `json`, `yaml`, `table`, `wide`; default text.

## Usage & Permissions
Root or nothing: the socket is `srw-rw---- root docker`, so the daemon forces `privileged: true` and this tool takes no `privileged` argument. It needs `docker/containers: {allowed: true}` in the caller's grant in `configs/mcp-sudo.yaml`; without it the tool is not even listed by `tools/list`.

Unlike `docker/logs`, `docker/exec` and `docker/manage`, it names no container, so it needs no `containers:` list — it lists every container on the host.

`linuxctl get docker containers` calls this tool (the bare `linuxctl get docker` resolves identically - `containers` is docker's bare-reachable default). For one container's own view, read `container://<name>/status`.
