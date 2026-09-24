---
sidebar_position: 4
---

# Permissions and Risks

:::danger Read this before granting anything
A grant in `mcp-sudo.yaml` gives **an AI agent** root on your host - an agent that follows instructions found in the files, web pages and logs it reads. Several grants that look narrow are, in practice, **full root**: write access to `/etc`, `services/manage`, unrestricted sysctl writes. Grant the least a task needs, limit it with `paths:`, `network:` and `sysctl:`, and assume that anything an agent may do, someone who controls its input may make it do.
:::

This page is about the defaults - what a user can do with no grant, and what each kind of grant adds - and about the grants that are riskier than they look. The syntax of every option is on [Sudo Privileges](./mcp-sudo).

## Who a call runs as

Every tool call runs in a separate worker process, as the **OS account with the same name as the MCP user** (`alice` in `users.yaml` → the Linux account `alice`):

- **Without `privileged: true`** - as that account, with its groups. The kernel decides what it may read, write or signal, exactly as for a shell of that account. `mcp-sudo.yaml` is not consulted, except for the `network:` limits below.
- **With `privileged: true`** - as root, but only for a tool granted to that user in `mcp-sudo.yaml`, and only within the grant's `paths:`/`sysctl:` limits. Anything else is refused before a worker starts.

So the OS account is the first permission, and it matters as much as the grants:

- An MCP user must not be a **root account**. mcpd refuses every call of a user whose account has uid 0 (it would be root without any grant), and `linuxctl create mcpd user` won't create one.
- **Groups count.** An account in `docker`, `lxd` or `disk` is root in all but name (it can start a privileged container, or read raw disks); `adm` and `systemd-journal` read every log. Create a dedicated account for each agent, in no such group: `useradd --system --no-create-home agent-web`.

## The defaults

| What the user has | What the agent can do |
|---|---|
| A token, **no entry** in `mcp-sudo.yaml` | Call every tool and resource **as its own OS account**: read what that account can read, change what it owns. No root. `daemon/reload-config` and the `devices://`/`kernel://` resources are refused. **`network/curl` and `network/ping` reach any address** - see below. |
| `allowed: true` on a tool **without a path** (`services/manage`, `processes/delete`, `kernel/system-control`, `logs/*`, ...) | That tool as root, **with any arguments**: any service, any process, any kernel parameter. |
| `allowed: true` on a **path tool** (`files/*`, `disks/usage`, `disks/free`) | Nothing, and the config is rejected: a path tool must list its `paths:`. `paths: ["/"]` for the whole filesystem - written out, so whoever reads the config sees it. |
| `allowed: true` + `paths: [/var/log]` | That tool as root **inside `/var/log` only** (`/var/log/../../etc` is judged as `/etc` and refused). Symlinks are not followed there, so one planted in `/var/log` can't lead out of it. |
| `network:` on `network/curl`/`network/ping` | Limits **every** call of the tool - privileged or not - to the allowed destinations. Without it: any destination. |
| `sysctl: write_keys:` on `kernel/system-control` | Root writes only to the listed keys. Without it: any key. Reads never need root. |
| `resources:` | Reading those resources as root: `"*"` for fixed ones (`devices://usb`), a path prefix for `file://` (`/var/log`), `""` for all of a template (`file://` → every file). |

### Examples

A user with only a token:

```yaml
# mcp-sudo.yaml: no entry for "reader"
```

| Call | Result |
|---|---|
| `files/read` `/etc/os-release` | ✅ world-readable |
| `files/read` `/etc/shadow` | ❌ `permission denied` - the kernel refuses the `reader` account |
| `files/read` `/etc/shadow`, `privileged: true` | ❌ `user reader is not authorized to run files/read on path /etc/shadow as root` |
| `processes/delete` another user's process, `privileged: true` | ❌ `Permission denied. Hint: You are not authorized to use 'privileged: true' for this tool in mcp-sudo.yaml` |
| `services/manage restart nginx` | ❌ refused by systemd: a non-root account may not manage system units |
| `network/curl http://169.254.169.254/` | ✅ **allowed** - no `network:` block |

A log reader:

```yaml
users:
  logs:
    privileged:
      tools:
        files/read:
          allowed: true
          paths: [/var/log]
        files/list:
          allowed: true
          paths: [/var/log]
        logs/journal-control:
          allowed: true
```

| Call | Result |
|---|---|
| `files/read` `/var/log/auth.log`, `privileged: true` | ✅ |
| `files/read` `/var/log/../../etc/shadow`, `privileged: true` | ❌ `user logs is not authorized to run files/read on path /var/log/../../etc/shadow as root` |
| `files/read` `/var/log/link-to-shadow` (a symlink), `privileged: true` | ❌ `refusing to follow a symbolic link` |
| `files/update` `/var/log/x`, `privileged: true` | ❌ not granted |

## Grants that amount to full root

Each of these lets an agent turn its grant into unrestricted root - by writing a file root executes, by starting a program as root, or by reading a secret. Treat them as "root on this host", and grant them only where that is acceptable.

| Grant | Why it is full root |
|---|---|
| `files/create` / `files/update` with `paths` covering `/etc`, `/root`, `/usr`, `/var/spool/cron`, `/etc/systemd`, or `/` | Write `/etc/sudoers.d/x`, a cron job, `/root/.ssh/authorized_keys`, a systemd unit, a binary on `PATH`. |
| `files/create` / `files/update` on **mcpd's own config directory** (`/etc/mcpd/configs`) | Rewrite `mcp-sudo.yaml` or `users.yaml` and grant itself everything at the next reload or restart. There is deliberately no MCP tool that edits them - don't hand one over through `files/*`. |
| `files/chmod` / `files/chown` on `/` or system directories | Make a copy of a shell setuid root, or `/etc/shadow` world-readable. |
| `files/read` on `/`, or `file://` with `""` | `/etc/shadow`, SSH host keys, mcpd's TLS private key (`/etc/mcpd/configs/tls/mcpd.key`), application secrets, other users' files. |
| `services/manage` | Start, stop, restart, enable any unit - stop `ssh` or `mcpd` itself; together with any write to `/etc/systemd`, run any program as root. |
| `kernel/system-control` **without** `sysctl.write_keys` | `kernel.core_pattern` and `kernel.modprobe` name programs the kernel runs as root; other keys can cut the host off the network. |
| `processes/delete` | Signal any process: `sshd`, `mcpd`, PID 1, a database mid-write. |
| MCP user whose OS account is in `docker`/`lxd`/`disk` | Root through the group, even with no grant at all (see above). |

### Also dangerous without any grant

- **`network/curl`, `network/ping`** connect *from the server*: to `localhost` services, the private network, the cloud metadata endpoint (`169.254.169.254`, which can hand out cloud credentials). A prompt-injected agent can be told to fetch an internal URL and pass the answer on. Give every user that doesn't need internal access a `network:` block with `deny_private: true`.
- **Reading logs and process lists** shows command lines, environment-derived arguments and log lines - sometimes with passwords or tokens in them.
- **The token is the user.** Whoever has it acts as that user. Keep TLS on (the default), give each agent its own user and token, and rotate a token (`linuxctl update mcpd user NAME`) whenever it may have leaked.

## Recipes

**Read-only diagnostics** - load, memory, disks, processes, services, logs readable by the account: no entry at all, plus a network limit.

```yaml
users:
  diag:
    privileged:
      tools:
        network/curl:
          network: { deny_private: true }
        network/ping:
          network: { deny_private: true }
```

**Log reader** - as above, plus root reads under `/var/log` and the journal (the log reader example).

**Web server operator** - its config and site files, reloading nginx; nothing else as root.

```yaml
users:
  web:
    privileged:
      tools:
        files/read:
          allowed: true
          paths: [/etc/nginx, /var/www, /var/log/nginx]
        files/update:
          allowed: true
          paths: [/etc/nginx/sites-available, /var/www]
        files/list:
          allowed: true
          paths: [/etc/nginx, /var/www, /var/log/nginx]
        services/manage:
          allowed: true      # any unit - see the table above; prefer this only on a host that is nginx and nothing else
        network/curl:
          network:
            allow: [127.0.0.1]   # its own sites via localhost
            deny_private: true
```

`services/manage` has no per-unit limit yet, so on a shared host this recipe is still "root" through it - leave it out there, and restart nginx yourself.

## Before granting - a checklist

1. Does the task need root at all? Most diagnostics don't: try the call without `privileged: true` first.
2. Is the MCP user a dedicated OS account, not root and not in `docker`, `lxd`, `disk`, `adm`?
3. Is every path tool limited to the directories the task needs - and none of them `/etc`, `/root`, `/usr`, `/etc/systemd`, `/etc/mcpd` for writes?
4. Is the grant on the "full root" list above? Then decide as if giving a root shell.
5. Does the user have `network:` limits on `network/curl`/`network/ping`?
6. Does `kernel/system-control` have `write_keys`, or no `allowed` at all?
7. After the change, read the reload's summary (`linuxctl edit mcpd config sudo` prints it; the daemon logs it as an `audit=true` line) and check it says what you meant.
