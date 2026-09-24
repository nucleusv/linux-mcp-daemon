---
sidebar_label: 'Daemon User Administration'
sidebar_position: 5
---

# Managing Daemon Users/Tokens: `linuxctl <verb> mcpd user`

The `mcpd` group edits the daemon's config files directly on disk - `configs/users.yaml` (users and token hashes) and `configs/mcp-sudo.yaml` (root grants) - and works even if the daemon isn't running. There is deliberately **no MCP tool that edits these files**: user/token administration is a privilege-escalation-relevant surface, so it stays with whoever has local access to the files (root, for an installed daemon), never with a remote agent.

After a change, `linuxctl` applies it to the running daemon by calling [`daemon/reload-config`](../mcp-api/tools/daemon/reload-config) (with `MCP_SERVER` and `MCP_TOKEN`, like any other command) - the daemon re-reads its files; nothing is written over the network. See [Users and Tokens](../configuration/users) for the file format.

```bash
linuxctl create   mcpd user alice [--set-token VALUE] [--grant TOOL,...] [--config-path DIR] [--no-reload]
linuxctl update   mcpd user alice [--set-token VALUE] [--config-path DIR] [--no-reload]   # rotates the token
linuxctl delete   mcpd user alice [--config-path DIR] [--no-reload]
linuxctl list     mcpd users      [--config-path DIR]
linuxctl describe mcpd user alice [--config-path DIR]
linuxctl edit     mcpd config [sudo|users|daemon] [--config-path DIR] [--no-reload]
```

- `--config-path` defaults to `./configs` (installed daemon: `/etc/mcpd/configs`).
- `--grant TOOL,...` lets the new user run those tools as root (`allowed: true` in `mcp-sudo.yaml`).
- `--no-reload` only edits the files; apply later with `linuxctl reload daemon`.

## Example - adding a user to a running daemon

```bash
$ sudo useradd --system --shell /usr/sbin/nologin alice
$ sudo -E /usr/local/bin/linuxctl create mcpd user alice --config-path /etc/mcpd/configs
Created mcpd user "alice".

Token (shown once - not stored in plaintext anywhere, save it now):
  863953e1c3a24cee64c4c2306af990da5fd9041bb8c9db5f0d48d478df5d860d

Next steps:
  1. Make sure an OS account "alice" exists on the mcpd host - each call runs as that account:
       useradd --system --shell /usr/sbin/nologin alice
     (for the container image, add it to the Dockerfile instead)
  2. Optionally grant root for specific tools in /etc/mcpd/configs/mcp-sudo.yaml - without grants "alice" can use
     every tool, but only as its own OS account, never as root

Asking mcpd at http://127.0.0.1:9091 to reload its config...
Reloaded configs/daemon.yaml, configs/users.yaml and configs/mcp-sudo.yaml.
Changes:
  user alice: added
```

`sudo -E` keeps your `MCP_SERVER`/`MCP_TOKEN` for the reload (the token's user needs `daemon/reload-config` granted - the installer's first user has it; on an install [upgraded from v0.1.0](../installation#upgrading) grant it first). Without them the files are still written, and `linuxctl` says how to apply them:

```
Not applied yet: no MCP_TOKEN to call mcpd with.
Apply by hand: linuxctl reload daemon (as a user granted daemon/reload-config), or restart mcpd
```

A rotated or deleted token stops working the moment the reload runs, and that user's open sessions are closed.

## Editing a config file by hand: `linuxctl edit mcpd config`

Like `visudo`: the file is copied, the copy opened in `$VISUAL` or `$EDITOR` (default `vi`; plain `sudo` drops your `EDITOR`, `sudo -E` keeps it), and on save checked with the same strict parser the daemon uses on reload. Only a valid file replaces the original (atomically, keeping its owner and mode), and then the daemon is reloaded:

```bash
$ sudo -E /usr/local/bin/linuxctl edit mcpd config sudo --config-path /etc/mcpd/configs
Saved /etc/mcpd/configs/mcp-sudo.yaml.

Asking mcpd at http://127.0.0.1:9091 to reload its config...
Reloaded configs/daemon.yaml, configs/users.yaml and configs/mcp-sudo.yaml.
Changes:
  grants alice: + files/list (root; paths [/var/log])
```

An invalid edit never reaches the real file - a misspelled key included:

```
mcp-sudo.yaml is not valid: yaml: unmarshal errors:
  line 17: field path not found in type config.ToolPrivilege
(e)dit again or (d)iscard the edit? [e] d
Discarded the edit - /etc/mcpd/configs/mcp-sudo.yaml left unchanged.
```

Consistent-but-suspicious edits are saved with a warning, e.g. `mcp-sudo.yaml has grants for "alice", which isn't an mcpd user`.

## Why `create`/`delete` touch both files atomically

Doing this by hand (editing `users.yaml` and `mcp-sudo.yaml` as two separate manual edits) is exactly how a stale privilege grant survives a user being "deleted" and later recreated under the same name - if only one file gets edited, the other file's old grants silently persist and apply to whoever gets that username next. `linuxctl create mcpd user` always writes a **fresh** `mcp-sudo.yaml` block (only the `--grant`ed tools), overwriting any stale leftover entry rather than merging with it, and `linuxctl delete mcpd user` always removes both files' entries together.

## What still needs to happen manually

A new user still needs a matching **OS account** before its calls can run - every tool call runs as that account. On a host: `useradd --system --shell /usr/sbin/nologin <username>`. For the container image: a `useradd` line in the `Dockerfile` and an image rebuild (the Kubernetes dev setup: `scripts/deploy.sh`, which also bakes the configs into the image - a reload can't change files inside a running image).

See [Users and Tokens](../configuration/users) for the salted-hash token storage and the trust-on-first-use OS identity pinning that guards against the OS account behind a username silently changing after the fact.
