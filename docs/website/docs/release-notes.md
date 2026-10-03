---
sidebar_position: 99
sidebar_label: 'Release Notes'
---

# Release Notes

What changed in each release, and what to do when upgrading. Every release, with its binaries, packages and the full commit list, is on [GitHub Releases](https://github.com/nucleusv/linux-mcp-daemon/releases).

## 0.5.0

**One breaking change: the Docker resource templates were renamed.** Everything else in this release needs no changes to your configs.

### Breaking: Docker resource templates are prefixed with `docker-`

`container://`, `image://` and `volume://` are now `docker-container://`, `docker-image://` and `docker-volume://`. `docker-network://` already had the prefix, so all four now match, and the bare nouns stay free. **There are no aliases**: the old names are `unknown resource`, and a user whose `resources:` grant in `mcp-sudo.yaml` still lists them cannot read the new ones.

| 0.4.1 | 0.5.0 |
|---|---|
| `container://{name}/{view}` | `docker-container://{name}/{view}` |
| `image://{name}/inspect` | `docker-image://{name}/inspect` |
| `volume://{name}/inspect` | `docker-volume://{name}/inspect` |
| `docker-network://{name}/inspect` | unchanged |

Rename the keys under `resources:` in each user's grant. This command does it, then restart or reload the daemon (the `docker/inspect` grant under `tools:` does not change):

```bash
sed -i -E 's#^(\s+)(container|image|volume)://:#\1docker-\2://:#' /etc/mcpd/configs/mcp-sudo.yaml
systemctl restart mcpd        # or: linuxctl reload daemon
```

The keyword forms of `linuxctl` (`linuxctl get docker container web-1 status`) are unchanged; only the explicit `linuxctl resource <uri>` form and raw `resources/read` calls use the new names. See [Docker templates](./mcp-api/resource-templates/Docker%20Container%20Introspection) and [mcp-sudo.yaml](./configuration/mcp-sudo).

### New: crontabs - `cron/manage` and `crontab://`

Read, list and replace crontabs: `linuxctl get crontabs`, `linuxctl update crontabs --content "$(cat mycron.txt)"`, and `linuxctl edit crontabs`, which opens `$EDITOR` like `crontab -e` and refuses to save if the crontab changed in the meantime. The `crontab://{user}/text` and `/info` resource template reads the same data.

- **Your own crontab needs no grant.** Another account's needs `privileged: true` and a rule naming it in your `cron/manage` grant: `users: {deploy: {view: true, edit: true}}`. Rules name accounts one by one (no wildcards), `edit` implies `view`, and `root` can be viewed but never edited.
- **It runs as the target account**, through the `crontab` command, so `cron.allow` and `cron.deny` still apply and the syntax is checked by `crontab` itself. A stale `if_match` hash is refused.
- **Writing a crontab schedules commands as that user, and the job outlives the session and the token.** mcpd stores the text exactly as given and does not inspect it; the named `edit` rule is the only control. Every write is audit-logged (size, lines, hash, never the content). Read the [crontab risks](./configuration/permissions-and-risks#crontabs) before granting `edit`.

See [`cron/manage`](./mcp-api/tools/cron/manage) and [`crontab://`](./mcp-api/resource-templates/Crontab).

### New: `timers/list` - systemd timers

Lists the systemd timers of the host with the unit each starts, its schedule, next and last run, and last result. Read-only; `services/list` never showed them. See [`timers/list`](./mcp-api/tools/timers/list).

### New: MCP annotations on every tool

`tools/list` now carries `title`, `readOnlyHint`, `destructiveHint`, `idempotentHint` and `openWorldHint` for all 48 tools, so a client can decide what needs a confirmation.

### Changed: all tool descriptions rewritten

Every description now says what the tool is for, when to use a sibling instead, its side effects and grants, how parameters interact and what comes back. Several descriptions that overstated what a tool does were corrected.

### Image: `cron` included

The container image now has the `crontab` command, so `cron/manage` works in the Docker and Kubernetes deployments. Inside a container a call sees the container's own crontabs.

### Fixed

- `timers/list` reported the next run of monotonic timers (`OnBootSec=`, `OnUnitActiveSec=`) as `never`.
- The crontab worker resolves the target account in its own namespace: a master running in a container numbers accounts differently from the host.

### Docs

Real READMEs for every resource template, the [`docker/manage`](./mcp-api/tools/docker/manage) page lists all seven actions, and the [command reference](./linuxctl/command-reference) is in alphabetical order with `timers` and `crontabs` as their own sections.

## 0.4.1

Upgrading from 0.4.0 needs no changes to your configs.

### Changed: every tool is listed to every user

In 0.4.0 the `docker/*` tools and `daemon/reload-config` were listed in `tools/list` only for users whose grant allowed them. Now every tool is listed for everyone, and a call the user's grant does not allow fails with an error that names the missing grant, before anything reaches the Docker socket:

```text
user testuser is not authorized to run docker/exec: the Docker socket is root-owned, so every docker/* tool needs `allowed: true` in its grant in mcp-sudo.yaml ...
```

A wrong call no longer looks like a typo (`no read target ...`), and `get mcp-api tools`, `linuxctl` tab completion and any client's tool list show the same surface to every user. What a user may *do* is unchanged - it is still decided by [mcp-sudo.yaml](./configuration/mcp-sudo). The four Docker resource templates were already listed to everyone; the tools now match them.

### New: SSE keepalive

An idle `/sse` stream now gets a `: ping` comment line every 20 seconds (`server.sse_keepalive_seconds` in [`daemon.yaml`](./configuration/daemon), read at startup). Bridges such as `mcp-remote` - what Claude Desktop uses - drop a stream that is silent for 300 seconds, reconnect with a new session and lose the calls in flight, for example while the agent waits for a human's approval. Clients ignore comment lines, so nothing else changes. Open sessions still survive a config reload; only the sessions of removed users, or users whose token changed, are closed.

### Tests

`tests/test_linuxctl.sh` runs against any target - local k8s by default, a VPS stand through environment variables - and has a docker section.

## 0.4.0

Upgrading from 0.3.5 needs no changes: existing configs, tools and their output are the same. The new Docker tools are refused until you grant them in `mcp-sudo.yaml`, and a `docker/*` tool without its grant is not listed in `tools/list`. The four Docker resource templates (`container://`, `image://`, `volume://`, `docker-network://`) are listed for every user, but reading one is refused without its own `resources:` grant.

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
