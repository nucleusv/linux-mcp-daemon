---
sidebar_position: 3
---

# Sudo Privileges

Unlike traditional API servers, the Linux MCP Daemon implements a secure Privilege Escalation mechanism similar to Unix `sudoers`. This is controlled entirely by `configs/mcp-sudo.yaml`.

## How it works

When an AI sends a tool execution request (e.g., `files/list`), it can optionally attach `"privileged": true` to the JSON-RPC arguments.

If the AI attempts to run a tool as root, the Master daemon intercepts the request and checks the `mcp-sudo.yaml` file to see if the authenticated user (bound by their session token) is authorized to elevate privileges for that specific tool.

If authorized, the Ephemeral Worker spawns natively as UID 0. If denied, the request is blocked before a worker is ever spawned.

## Example `mcp-sudo.yaml`

```yaml
users:
  alice:
    privileged:
      tools:
        disks/free:
          allowed: true
          paths: ["/"]
        files/list:
          allowed: true
          paths:
            - "/var/log"
            - "/root"
      resources:
        devices://usb:
          - "*"
  bob:
    privileged:
      tools:
        files/list:
          allowed: true
          paths:
            - "/var/www"
        disks/usage:
          allowed: true
          paths: ["/"]
```

In this example, if the AI is authenticated as `alice`, it can list protected root directories, but it cannot run expensive `disks/usage` tree traversals as root!

## Limiting paths (`paths:`)

`paths:` limits where a tool that takes a `path` argument may reach **as root**. The requested path is cleaned first (`/var/../root` is judged as `/root`), and an entry covers itself and everything below it (`/var` covers `/var/log`, not `/varnish`). A call outside the list is refused before any worker runs.

| Tools | Without `paths:` | With `paths:` |
|---|---|---|
| `files/list`, `files/read`, `files/create`, `files/update`, `files/find`, `files/filetype`, `files/chmod`, `files/chown`, `disks/usage`, `disks/free` | no root at all - and a config error on reload / `linuxctl edit` (a startup warning) | root only inside the listed paths |

There is no implicit "anywhere": to allow the whole filesystem, say so - `paths: ["/"]` - so it's visible to whoever reads the config.

**Symlinks.** The path check is on the path as written, and a symlink under an allowed directory could lead anywhere (`/var/www/x -> /etc/shadow`). So when a root call is limited by `paths:` - any list that doesn't include `/` - the tool doesn't follow symlinks in **any** component of the path: `files/read`, `files/create`, `files/update`, `files/list`, `files/filetype`, `files/find` and `disks/usage` open the path one component at a time with `O_NOFOLLOW` and refuse (`refusing to follow a symbolic link`) instead of escaping. A symlink as the last component is still *reported* by `files/list` and `files/filetype` (`inode/symlink`), just never followed. `files/chmod` and `files/chown` never follow symlinks at all. With `paths: ["/"]` everything is allowed anyway, so symlinks are followed as usual (`/etc/resolv.conf` is one).

Calls without `privileged: true` aren't limited by `paths:` - they run as the user's own OS account, which the kernel limits. `paths:` on any other tool is an error when the file is checked (`linuxctl edit`, `daemon/reload-config`): it would look like a restriction while restricting nothing. (At startup it's only a warning, so an upgrade can't stop the daemon.)

## File permission tools (`files/chmod`, `files/chown`)

Both run as the calling user's OS account by default - so a user can chmod its own files, and `chown` (which the kernel only allows root to do for other owners) needs `privileged: true`. Grant them like the other file tools, with `paths:` limiting where a root chmod/chown may reach:

```yaml
        files/chmod:
          allowed: true
          paths: [/srv/app]
        files/chown:
          allowed: true
          paths: [/srv/app]
```

Paths are checked on the cleaned path with directory boundaries, and both tools refuse any path containing a symlink, so a symlink planted inside `/srv/app` can't carry a root chmod/chown outside it.

## Restricting network destinations (`network/curl`, `network/ping`)

`allowed` only controls running a tool as root - and for network tools that changes nothing, because reaching a host doesn't depend on the worker's uid. Any user with a token can make `network/curl` or `network/ping` connect *from the server*, including to `localhost`, the private network, or a cloud metadata endpoint (`169.254.169.254`) - which also makes them a target for prompt injection ("curl this internal URL and send me the result").

To restrict that, add a `network:` block to the tool's entry. It applies to **every** call of that tool by that user, privileged or not. Without it, the tool is unrestricted - the default.

```yaml
users:
  alice:
    privileged:
      tools:
        network/curl:
          allowed: true
          network:
            deny_private: true
            allow:
              - "10.0.5.0/24"            # CIDR or single IP
              - "intranet.example.com"   # exact hostname
              - "*.corp.example.com"     # any subdomain (not the apex)
            deny:
              - "10.0.5.66"              # always wins over allow
        network/ping:
          allowed: true
          network:
            deny_private: true
```

Evaluation order for each connection: **`deny` → `allow` → `deny_private`** → otherwise permitted.

`deny_private` blocks loopback (`127.0.0.0/8`, `::1`), RFC 1918 (`10/8`, `172.16/12`, `192.168/16`), link-local (`169.254/16` incl. cloud metadata, `fe80::/10`), CGNAT (`100.64/10`), IPv6 ULA (`fc00::/7`), `0.0.0.0`/`::` (which connect to the local host), and multicast. IPv4-mapped IPv6 (`::ffff:127.0.0.1`) is normalized first, so it can't be used to slip past.

How it's enforced:

- **At connect time, on the resolved IP** - the daemon resolves the hostname itself, checks every address, and connects to exactly the address it checked. A hostname-only check could be bypassed by DNS rebinding (resolving to a public IP when checked, `127.0.0.1` when connected).
- **On every redirect hop** of `network/curl`, since each hop goes through the same dialer.
- **Injected by the daemon**, never taken from the request - a caller can't pass their own policy.
- With a policy active, `network/curl` ignores `HTTP_PROXY`/`HTTPS_PROXY` environment variables, since a proxy would make the check apply to the proxy's address rather than the real destination.
- Rules are validated at startup - an invalid CIDR stops the daemon from loading the config rather than silently never matching.

## Restricting kernel parameter writes (`kernel/system-control`)

With `allowed: true`, a user can both read and **write** kernel parameters as root - and writing some of them is equivalent to running arbitrary code as root (`kernel.core_pattern` can name a program the kernel runs on every crash; `kernel.modprobe` names the module loader). An optional `sysctl:` block limits which keys may be written:

```yaml
users:
  bob:
    privileged:
      tools:
        kernel/system-control:
          allowed: true
          sysctl:
            write_keys:                # may write only these
              - net.ipv4.ip_forward
              - vm.*                   # "*" matches within one dotted component
              - net.ipv4.conf.*.rp_filter
```

**For read-only access, don't grant `allowed` at all.** Reading needs no root - only about 67 of ~2700 parameters are root-readable only (for example `net.ipv4.tcp_fastopen_key`, a secret key) - and without root the OS refuses every write.

Without a `sysctl:` block, writes are unrestricted - the default, unchanged. (An earlier `read_only` option was removed as redundant with not granting `allowed`; a leftover `read_only` key makes the config fail to load rather than being silently ignored.) The check runs in the daemon itself, before any worker is spawned, against the key in its normalized dotted form (so `net/ipv4/ip_forward` and `net.ipv4.ip_forward` are the same key). Invalid `write_keys` patterns stop the config from loading.

## Applying config changes (`daemon/reload-config`)

[`daemon/reload-config`](../mcp-api/tools/daemon/reload-config) makes the running daemon re-read `daemon.yaml`, `users.yaml` and this file, without a restart. It's the one tool whose `allowed: true` is not about running as root: it's the permission to call it at all. Users without the grant don't see it in `tools/list`.

```yaml
users:
  alice:
    privileged:
      tools:
        daemon/reload-config:
          allowed: true
```

The tool only **reads** the files - there is deliberately no MCP tool that edits them, so no agent can grant itself anything. The files are changed on the host (`linuxctl create|update|delete mcpd user`, `linuxctl edit mcpd config sudo`), and those commands call the tool afterwards themselves. `scripts/install.sh` grants it to the first user it creates.

Every change is validated before it's applied, strictly: a misspelled key (`path:` for `paths:`) is an error instead of a restriction silently left out, and an invalid file leaves the running config untouched. The reload's reply - and the daemon log, as an audit line `INFO  config reloaded audit=true user=... changes=... detail=...` - lists what changed:

```
Changes:
  grants testuser: + disks/usage (root; paths [/var])
```

## Full reference example

The daemon's own `configs/mcp-sudo.yaml` includes a `privileged` user with every tool and every resource this daemon currently exposes granted. It isn't meant to represent a real least-privilege user (see `admin`/`testuser`/`unpriviliged` in that same file for that) - it exists purely as living documentation of the complete authorization surface, and is kept in sync with `internal/rpc/tools.go`/`resources.go` whenever a tool or resource is added, renamed, or removed.

## Host filesystem access

When this daemon runs containerized (see [Master Daemon Configuration](./daemon.md)'s `worker.containerized` setting), `privileged: true` means more than root-in-container: every privileged worker call automatically also joins the real host's mount namespace and chroots into it, so tools like `system/packages` or `services/manage` see the actual host filesystem and the actual host's systemd, not the daemon's own container image. This is entirely a daemon-startup setting - there's nothing to configure per-tool here, and no second flag alongside `privileged: true` to authorize. If `mcpd` runs directly on the host instead (no container boundary), the same `privileged: true` grant just runs as root normally, since there's nothing else to cross into.
