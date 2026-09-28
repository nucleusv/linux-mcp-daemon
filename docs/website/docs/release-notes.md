---
sidebar_position: 99
sidebar_label: 'Release Notes'
---

# Release Notes

What changed in each release, and what to do when upgrading. Every release, with its binaries, packages and the full commit list, is on [GitHub Releases](https://github.com/nucleusv/linux-mcp-daemon/releases).

## 0.4.0

Upgrading from 0.3.5 needs no changes: existing configs, tools and their output are the same. The new Docker tools stay off, and out of `tools/list`, until you grant them in `mcp-sudo.yaml`.

### New: Docker tools

Read and manage containers on the host through the Docker Engine API over its unix socket - no `docker` CLI, no new dependency. Docker must already be installed and running; mcpd never installs it.

| | |
|---|---|
| Read | [`docker/containers`](./mcp-api/tools/docker/containers), [`docker/images`](./mcp-api/tools/docker/images), [`docker/volumes`](./mcp-api/tools/docker/volumes), [`docker/networks`](./mcp-api/tools/docker/networks), [`docker/logs`](./mcp-api/tools/docker/logs) |
| One object | `container://{name}/{view}`, `image://{name}/inspect`, `volume://{name}/inspect`, `docker-network://{name}/inspect` |
| Act | [`docker/manage`](./mcp-api/tools/docker/manage) (start, stop, restart, kill, pause, unpause, remove), [`docker/exec`](./mcp-api/tools/docker/exec), [`docker/prune`](./mcp-api/tools/docker/prune) |

```bash
linuxctl get docker containers
linuxctl get docker network bridge
linuxctl restart docker web-1
linuxctl exec docker web-1 ls /usr/share/nginx/html
```

Every `docker/*` call runs as root (the socket is root-owned) and is refused without an explicit grant. The tools that name a container take a `containers:` list of name or ID globs - one list per tool, so `exec` can be narrower than `manage` - and `prune` takes a `prune:` list of the kinds it may reclaim; a grant with no list refuses everything. Details and examples: [mcp-sudo.yaml](./configuration/mcp-sudo).

### Fixed in `linuxctl`

- **`get docker <keyword> <name>` read the wrong thing.** `get docker network bridge` fell through to the group's default tool; it now reads that one object (also `container`, `volume`, `image`).
- **A wrong word is an error.** `linuxctl get docker networkz` printed the container list and exited 0; it now stops with `no read target "networkz" in group "docker"`, exit 1.
- **Extra words are reported.** `describe` and template `get` reads print `Warning: N extra argument(s) ignored: ...` instead of dropping them.
- **`--` ends flags.** `linuxctl exec docker web-1 -- echo hello` lost `echo` and ran `hello`; everything after `--` now reaches the container as-is.
- **Tab completion** for `get docker` offers `containers` and the `container` / `network` / `volume` / `image` keywords.

## 0.3.5

Upgrading from 0.3.4 needs no changes: configs, tools and their output are the same.

### New: stdio mode

`mcpd stdio` serves MCP over stdin/stdout, for clients that start a server as a local process - no daemon, no port, no token, no TLS:

```bash
claude mcp add linux -- mcpd stdio                                  # this machine
claude mcp add web1 -- ssh admin@web1 mcpd stdio                    # a remote server over SSH, no open port
claude mcp add app -- docker exec -i app mcpd stdio --user app      # inside a container
```

Tool calls run as the OS user who started mcpd and **never as root**: `privileged: true` is refused and `mcp-sudo.yaml` is not read. Started as root (`docker exec`, a catalog's build), mcpd requires `--user NAME` and runs each call as that account. When to use it and when the network daemon is the better fit: [Stdio mode](./configuration/stdio-mode).

### Fixed

- **Notifications are no longer answered.** A message without an `id` - `notifications/cancelled`, for example - used to get a `Method not found` reply with `id: null`. JSON-RPC forbids replying to a notification; over stdio a client would read it as an unexpected message.

### Elsewhere

- mcpd is listed in the official [MCP Registry](https://registry.modelcontextprotocol.io/v0/servers?search=io.github.nucleusv/linux-mcp-daemon) as `io.github.nucleusv/linux-mcp-daemon`, and on [Glama](https://glama.ai/mcp/servers/nucleusv/linux-mcp-daemon).
