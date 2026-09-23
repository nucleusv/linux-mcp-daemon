---
sidebar_label: 'Daemon User Administration'
sidebar_position: 5
---

# Managing Daemon Users/Tokens: `linuxctl <verb> mcpd user`

Unlike every other command in this section, `mcpd` never talks to the daemon over the network - it reads and writes `configs/daemon.yaml` + `configs/mcp-sudo.yaml` directly on disk, needs no `-token`/`-server` flag, and works even if the daemon isn't running. This is intentional: user/token administration is a privilege-escalation-relevant surface, kept off the network `tools/call` path entirely. See the [Daemon Configuration](../configuration/daemon) page for full detail (token hashing, why `create`/`delete` touch both files atomically, and the OS-account step a new user still needs before a rebuild).

```bash
linuxctl create   mcpd user alice [--set-token VALUE] [--config-path DIR]   # DIR defaults to ./configs
linuxctl delete   mcpd user alice [--config-path DIR]
linuxctl update   mcpd user alice [--set-token VALUE] [--config-path DIR]   # rotates the token
linuxctl list     mcpd users      [--config-path DIR]
linuxctl describe mcpd user alice [--config-path DIR]
```

## Example - creating a user and seeing the one-time token

```bash
$ linuxctl create mcpd user alice
Created mcpd user "alice".

Token (shown once - not stored in plaintext anywhere, save it now):
  863953e1c3a24cee64c4c2306af990da5fd9041bb8c9db5f0d48d478df5d860d

Next steps:
  1. Add a matching OS account (useradd -m -s /bin/bash alice in the Dockerfile) - workers run
     as a real OS user via user.Lookup(), so this account must exist before alice can make any call.
  2. Grant tools/resources for "alice" in configs/mcp-sudo.yaml (currently empty - denies everything by default)
  3. Run scripts/deploy.sh to rebuild and apply
```

## Why `create`/`delete` touch both files atomically

Doing this by hand (editing `daemon.yaml` and `mcp-sudo.yaml` as two separate manual edits) is exactly how a stale privilege grant survives a user being "deleted" and later recreated under the same name - if only one file gets edited, the other file's old grants silently persist and apply to whoever gets that username next. `linuxctl create mcpd user` always writes a **fresh, empty** `mcp-sudo.yaml` block, overwriting any stale leftover entry rather than merging with it, and `linuxctl delete mcpd user` always removes both files' entries together.

## What still needs to happen manually

A `create` (or a `token_hash`-migrated legacy user) still needs, before it can actually make any call:

1. A matching OS account in the `Dockerfile` (`useradd -m -s /bin/bash <username>`) - `linuxctl create` reminds you of this, but doesn't do it for you, since the account only becomes real after an image rebuild.
2. Tool/resource grants in `mcp-sudo.yaml` (a fresh user starts with none - denies everything by default).
3. `scripts/deploy.sh` to rebuild and redeploy.

See the [Daemon Configuration](../configuration/daemon) page for the salted-hash token storage format and the trust-on-first-use OS identity pinning that also guards against the OS account behind a username silently changing after the fact.
