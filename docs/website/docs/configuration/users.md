---
sidebar_position: 2
---

# Users and Tokens

`configs/users.yaml` lists who may connect to mcpd and with which bearer token. It's the one config file holding secrets (token hashes), so it's kept apart from [`daemon.yaml`](./daemon), readable by root only (`0600`), and written by `linuxctl` rather than by hand.

```yaml
users:
  - username: "alice"
    token_salt: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"
    token_hash: "9f8e7d6c5b4a...  # sha256(salt + plaintext token), hex"
    created_at: "2026-09-23T00:28:15Z"
    pinned_uid: "1001"   # written by mcpd itself - see below
    os_uid: "1001"
  - username: "bob"
    token: "secret456"   # legacy plaintext form - still supported; `linuxctl update mcpd user bob` migrates it
```

- **`username`** binds the token to the user's grants in [`mcp-sudo.yaml`](./mcp-sudo), **and** must be a real OS account: every tool call runs as that account (`SpawnWorker` resolves it with `user.Lookup()`). A user without grants can still call every tool - as their own OS account, never as root.
- **`token_salt` + `token_hash`**: `sha256(salt + token)`, hex. The token itself is stored nowhere - it's printed once when created or rotated. **Don't hand-write a `token_hash`**: the salt must be freshly random per user, and a hash that doesn't exactly match simply never authenticates. The daemon also accepts the legacy plaintext `token` field so existing accounts don't need a hard cutover.
- **`pinned_uid` / `os_uid`**: trust-on-first-use OS identity pinning, written by mcpd on a user's first successful call. If the username later resolves to a different UID (the OS account was deleted and recreated), mcpd refuses that user's calls until `linuxctl update mcpd user <name>` re-provisions it. Never set these by hand.

## Managing users

```bash
linuxctl create   mcpd user alice [--grant TOOL,...]   # random token, printed once
linuxctl update   mcpd user alice                      # rotate the token
linuxctl delete   mcpd user alice                      # from users.yaml and mcp-sudo.yaml together
linuxctl list     mcpd users
linuxctl edit     mcpd config users                    # hand edit, validated before saving
```

Each of these edits the files locally (`--config-path`, default `./configs`) and then applies the change to the running daemon through [`daemon/reload-config`](../mcp-api/tools/daemon/reload-config): the new user can connect at once, and a rotated or deleted token stops working at once (its open sessions are closed). Full reference: [Daemon User Administration](../linuxctl/mcpd-admin).

## Upgrading from `users:` in `daemon.yaml`

Older configs list users in `daemon.yaml`. mcpd still reads them from there (and logs a warning) as long as there is no `users.yaml`. The next `linuxctl create|update|delete mcpd user` - or `linuxctl edit mcpd config users` - moves the list to a new `users.yaml` (mode `0600`) and removes it from `daemon.yaml`. Users in both files is an error.
