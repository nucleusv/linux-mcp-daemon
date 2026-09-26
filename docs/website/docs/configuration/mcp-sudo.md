---
sidebar_position: 3
---

# Sudo Privileges

Unlike traditional API servers, the Linux MCP Daemon implements a secure Privilege Escalation mechanism similar to Unix `sudoers`. This is controlled entirely by `configs/mcp-sudo.yaml`.

:::warning
Some grants that look narrow amount to full root on the host (writes to `/etc`, `services/manage`, sysctl writes without `write_keys`). Before granting anything, read [Permissions and Risks](./permissions-and-risks): what a user can do with no grant, what each grant adds, and least-privilege recipes.
:::

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

It grants every tool as root, the whole filesystem to the path tools (`paths: ["/"]`) and every resource - **full root on the host**: a list of names to copy from, not a set of grants to give anyone (see [Permissions and Risks](./permissions-and-risks)). `scripts/check_docs.sh` fails when this copy and the file differ.

<details>
<summary><b>The <code>privileged</code> block of <code>configs/mcp-sudo.yaml</code></b> - every tool and resource</summary>

{/* reference-privileged:start */}
```yaml
users:
  privileged:
    privileged:
      tools:
        # What privileged: true adds per tool. ⚠ = amounts to full root on the host.
        auth/sudo-rules:  # read the host's sudoers rules
          allowed: true
        daemon/reload-config:  # not root: the permission to call it at all (re-reads the configs)
          allowed: true
        cpu/list:  # root adds little: readable by any user
          allowed: true
        cpu/load-average:  # root adds little: readable by any user
          allowed: true
        disks/free:  # statfs any path under paths
          allowed: true
          paths:
            - /
        disks/health:  # SMART data (smartctl needs root)
          allowed: true
        disks/list:  # containerized: the host's mount points
          allowed: true
        disks/mounts:  # containerized: the host's mount table
          allowed: true
        disks/partitions:  # partition tables of the host's disks
          allowed: true
        disks/performance:  # root adds little: /proc/diskstats is readable by any user
          allowed: true
        disks/usage:  # measure directories other users can't read
          allowed: true
          paths:
            - /
        files/create:  # ⚠ create any file under paths - full root if they cover /etc, /root, /usr
          allowed: true
          paths:
            - /
        files/filetype:  # inspect files other users can't read
          allowed: true
          paths:
            - /
        files/chmod:  # ⚠ change any mode under paths (setuid, /etc/shadow)
          allowed: true
          paths:
            - /
        files/chown:  # ⚠ change any owner under paths
          allowed: true
          paths:
            - /
        files/find:  # search directories other users can't read
          allowed: true
          paths:
            - /
        files/list:  # list directories other users can't read
          allowed: true
          paths:
            - /
        files/read:  # ⚠ read any file under paths (/etc/shadow, keys, mcpd's TLS key)
          allowed: true
          paths:
            - /
        files/update:  # ⚠ overwrite or append to any file under paths - full root if they cover /etc
          allowed: true
          paths:
            - /
        kernel/system-control:  # ⚠ write any kernel parameter unless sysctl.write_keys limits it
          allowed: true
          # Optional write restrictions (checked before any worker runs).
          # Absent = unrestricted. Example:
          # sysctl:
          #   write_keys: ["net.ipv4.ip_forward", "vm.*"]  # only these keys
        logs/dmesg:  # the kernel log, if dmesg is restricted to root
          allowed: true
        logs/logins:  # login records (wtmp/btmp)
          allowed: true
        logs/journal-control:  # every unit's journal
          allowed: true
        memory/usage:  # root adds little: readable by any user
          allowed: true
        network/arp:  # root adds little: readable by any user
          allowed: true
        network/connections:  # the owning process of every socket
          allowed: true
        network/curl:  # root doesn't change what it can reach - limit that with network:
          allowed: true
          # Optional per-tool destination restrictions (network/curl and
          # network/ping). Unlike `allowed` (root only), `network:` applies
          # to every call of the tool. Absent = unrestricted. Example:
          # network:
          #   deny_private: true        # loopback, 10/8, 172.16/12, 192.168/16, 169.254/16, CGNAT, ULA, ...
          #   allow: ["10.0.5.0/24", "intranet.example.com", "*.corp.example.com"]
          #   deny: ["10.0.5.66"]       # always wins over allow
        network/nslookup:  # root adds little
          allowed: true
        network/ping:  # root doesn't change what it can reach - limit that with network:
          allowed: true
        network/trace-path:  # traceroute as root
          allowed: true
        processes/delete:  # ⚠ signal any process: sshd, mcpd, PID 1
          allowed: true
        processes/list:  # every user's processes in full
          allowed: true
        processes/top:  # every user's processes in full
          allowed: true
        services/list:  # systemd's private socket instead of the system bus
          allowed: true
        services/manage:  # ⚠ start, stop, restart, enable any unit
          allowed: true
        # These three are never callable directly via tools/call (they're
        # not in tools.go's schema/standardWorkers) - they're the internal
        # worker names behind service://, file://, and process:// resource
        # templates. SpawnWorker independently re-checks CanRunAsRoot()
        # against this exact name whenever a resource template requests
        # privileged access, so these entries are required for privileged
        # reads of those resources to work, even though the name itself is
        # never a valid tools/call target.
        services/status:  # internal: the worker behind service://
          allowed: true
        files/content:  # internal: the worker behind file://
          allowed: true
        # Worker behind file:///{path}/stat - a privileged file:// read needs
        # both the resource grant and this one (see ARCHITECTURE.md).
        files/stat:  # internal: the worker behind file:///{path}/stat
          allowed: true
        processes/read:  # internal: the worker behind process://
          allowed: true
        system/os-release:  # containerized: the host's OS, not the image's
          allowed: true
        system/packages:  # containerized: the host's packages
          allowed: true
        users/list:  # containerized: the host's users
          allowed: true
      resources:
        os://uname:
          - "*"
        os://release:
          - "*"
        system://hostname:
          - "*"
        system://timezone:
          - "*"
        system://locale:
          - "*"
        network://interfaces:
          - "*"
        network://routes:
          - "*"
        devices://usb:
          - "*"
        devices://pci:
          - "*"
        devices://dmi:
          - "*"
        kernel://modules:
          - "*"
        # Unlike the exact-match resources above (where the code always
        # compares against the literal sentinel "*"), these are genuinely
        # prefix-matched against a real value (a path, a service name, a
        # PID) - so "grant everything" means an empty-string prefix, which
        # every value starts with. A literal "*" here would only grant
        # access to a resource actually named "*", never allowing any real
        # value through.
        file://:  # ⚠ "" = read any file as root
          - ""
        service://:
          - ""
        process://:
          - ""
```
{/* reference-privileged:end */}

</details>

## Host filesystem access (containers)

Two different things are called "privileged" - don't mix them up:

| | Docker's `--privileged` | `privileged: true` in a call |
|---|---|---|
| Where | `docker run`, once, when the container starts | the arguments of a `tools/call` (`--privileged true` in `linuxctl`) |
| Set by | whoever installs mcpd | the agent or user making the call |
| Means | give the container kernel capabilities (switch namespaces, chroot) | "run this call as root" |
| Checked | no - a Docker flag | yes - against this file, per user and tool |

When mcpd runs in a container, its own filesystem is the image's: its `/etc`, its packages, its `/etc/os-release`. A call made with `privileged: true` - by a user granted that tool here - runs as root **and switches to the host**: the worker joins the mount namespace of the host's PID 1 and chroots into its root, so it reads the host's files and talks to the host's systemd. A call without it stays in the container, as the user's own account. On the test host:

```text
without privileged:  PRETTY_NAME="Ubuntu 24.04.5 LTS"   <- the container image
with privileged:     PRETTY_NAME="Ubuntu 24.04 LTS"     <- the host
```

Switching to the host needs the container started with **`--privileged --pid host`** (see [the container install](../installation#container-image)); the image sets `worker.containerized: true`, which turns the switch on. What a privileged call does, by how the container was started (captured live):

| Container started with | A call with `privileged: true` |
|---|---|
| `--privileged --pid host` | runs as root **on the host** |
| `--pid host`, no `--privileged` | fails: `cannot switch to the host's filesystem - is the mcpd container running with --privileged --pid host? (unshare(CLONE_FS): operation not permitted)` |
| neither | ⚠️ runs as root **inside the container**, without an error: it sees the image (`Ubuntu 24.04.5 LTS`, the image's `/root`), not the host. mcpd only logs a warning at startup (`daemon.yaml sets worker.containerized: true, but this process does not appear to be in a separate mount namespace...`) |

So start the container exactly as the install shows, and check once that a privileged call reports the host: `linuxctl get system os-release --privileged true`.

When mcpd runs directly on the host (systemd), there's nothing to switch to: `privileged: true` simply runs the call as root.
