# Linux Admin Roadmap

Originally written 2026-09-22, updated 2026-09-23 as items were built. Verified directly against `internal/rpc/tools.go` and `internal/rpc/resources.go` (not `plan/tools/*.md`, which CLAUDE.md already flags as using stale pre-rename names). This supersedes those files as the planning reference; they're left in place as historical record but shouldn't be trusted for current tool names.

The lens here is deliberately different from `plan/tools/*.md`'s "which Linux CLI does this replace" checklist: this is organized by what a Linux administrator actually needs to *do* day to day, since that's what determines whether this daemon is useful as a real ops tool versus a read-only dashboard.

## Current state

**33 tools**: `cpu/list`, `cpu/load-average`, `disks/{free,usage,list,mounts,health,partitions,performance}`, `files/{list,read,create,update,find,filetype}`, `network/{connections,curl,nslookup,ping,arp,trace-path}`, `processes/{list,delete}`, `services/{list,manage}`, `logs/{dmesg,journal-control,logins}`, `kernel/system-control`, `system/{os-release,packages}`, `users/list`, `auth/sudo-rules`.

**12 static resources**: `os://{uname,release}`, `system://{hostname,timezone,locale}`, `network://{interfaces,routes}`, `devices://{usb,pci,dmi}`, `kernel://modules`.

**6 resource templates**: `file://{path}`, `devices://{type}`, `network://interfaces/{name}`, `service://{name}/status`, `disks://{name}/stats`, `process://{pid}/{target}` (targets: `status`, `cmdline`, `environ`, `limits`, `open_files`).

In short: strong read-side coverage of hardware, processes, disks, network state, services, mounts, users, and login history is now in place. The gaps below are mostly on the *change something* side.

## What's missing, by admin workflow

### 1. Users & access management — ✅ read side done (`users/list`, `logs/logins`)
Was the biggest gap; onboarding/access-review/incident-response read visibility now exists. `users/list` (native `/etc/passwd`+`/etc/group` parsing, never reads `/etc/shadow`) and `logs/logins` (wraps `last`/`lastb` rather than hand-parsing the binary `wtmp`/`btmp` struct format - confirmed during build that this daemon's real host has neither the log files nor the binaries installed at all, a genuine property of this minimal environment, not a bug) are both shipped and worker-routed (see the real host with `privileged: true`).

Still open, and these are real mutations, so they're tools:
- `users/create`, `users/delete`, `users/modify` - almost certainly needs `useradd`/`usermod`/`userdel` wrapping rather than hand-editing `/etc/passwd` (password hashing, home dir creation, `/etc/shadow` locking are exactly the kind of "native reimplementation is brittle" case `ARCHITECTURE.md` already has precedent for)
- `auth/ssh-keys` - read/append `~/.ssh/authorized_keys` (reuses `files/*` machinery conceptually, but path resolution per-user needs care)

### 2. Scheduled tasks — nothing exists today
- `cron/list`, `cron/create`, `cron/delete` - either parse `/var/spool/cron/crontabs/*` directly, or wrap `crontab -l`/`-e` per-user (direct file access is simpler but skips `cron`'s own reload signaling)
- systemd timers: `services/list` may already surface `.timer` units generically since it lists all systemd units, not just `.service` - worth verifying before building anything new here rather than assuming a gap

### 3. Firewall — read-only doesn't exist, write definitely doesn't
- `network/firewall-rules` (read) - `iptables-save`/`nft list ruleset`, both text-parseable
- `network/firewall-rules` (write) - add/remove rules; genuinely dangerous (a bad rule can lock out the daemon's own management access), so this is a strong candidate for the dry-run/confirmation governance pattern from day one, not bolted on after

### 4. Package management — read exists (`system/packages`), install/remove doesn't
- `system/package-install`, `system/package-remove` - same dpkg/apk/rpm detection `system/packages` already does; rpm-based install is a similar documented gap (would need `apt`/`apk` wrapping since there's no safe native "install" primitive - this is squarely a `ARCHITECTURE.md`-style CLI-wrapping exception, not something to hand-roll)

### 5. Mounts & filesystem — ✅ read side done (`disks/mounts`)
`disks/list` covers block devices; `disks/mounts` (shipped) covers what's actually mounted where, via `/proc/thread-self/mounts`. Automatically sees the real host's mounts with `privileged: true` when `mcpd` runs containerized.
- `disks/mount`, `disks/unmount` - still open: write actions, need care around namespace semantics (mounting inside the container's namespace vs. the real host's)
- `/etc/fstab` management - arguably just `files/read`+`files/update` already cover this generically; may not need a dedicated tool

### 6. System configuration — ✅ read side done, as resources not tools
Corrected from the original plan below: `hostname`/`timezone`/`locale` are configurable system-wide *properties*, not lists/actions - per the tools-vs-resources split (tools: lists, dynamic queries, create/update/delete; resources: an object's properties/status), the read side belongs under `system://`, not as tools. Shipped: `system://hostname` (moved from `os://hostname` - it was never an OS/kernel build fact like `os://uname`/`os://release`), `system://timezone`, `system://locale`. All three are read directly in the master daemon process (a narrow, documented exception - see each package's README) rather than worker-routed, so they report this daemon's own container's values, not the real host's, even when `worker.containerized: true`.

Still open, and these genuinely are tools (they're actions/mutations, not property reads):
- `system/hostname` (set) - `hostnamectl` equivalent
- `system/timezone` (set) - `timedatectl` equivalent
- `system/locale` (set) - `localectl` equivalent

Worth deciding when built: should the write side also be worker-routed (to reach the real host via `privileged: true` + `worker.containerized`), unlike their read-side resource counterparts? Probably yes, since setting the container's own hostname/timezone/locale rather than the real host's would be a fairly useless mutation for an admin tool - but that's a real design choice to make explicitly, not inherit by default.

### 7. Security & auditing — minimal today
Currently the only "security" surface is that `devices://*`/`kernel://*` resources require privilege. Nothing proactive exists.
- `security/suid-scan` - find SUID/SGID binaries system-wide (`find / -perm -4000`, natively via `filepath.WalkDir` + mode bits - no external binary needed, same technique `disks/usage` already uses)
- `security/failed-logins` - failed SSH/sudo attempts via `journal-control`/`auth.log` parsing
- `security/updates-available` - pending package updates (`apt list --upgradable` equivalent) - natural pairing with `system/packages`
- Open-ports-to-process mapping: check whether `network/connections` already correlates listening sockets to owning PIDs before assuming this needs new work

### 8. Container/Kubernetes awareness — genuinely relevant given actual deployment
This daemon's own real deployment (this session's whole environment) is a Kubernetes node, and today it has zero visibility into containerd/Kubernetes state despite running right next to it. An admin using this daemon *on a k8s node* would reasonably expect to inspect pods/containers the same way they inspect processes.
- `containers/list` - talk to containerd's own gRPC socket (kernel-first-equivalent: native containerd API client, not shelling to `crictl`)
- Scope this as a deliberate, separate decision rather than assuming it belongs in "generic Linux admin" - it's specific to this deployment context, not every install of this daemon runs on a k8s node

### 9. Process control — read exists, write doesn't
This session added `limits` (read) to `process://{pid}/{target}`; there's no write-side equivalent.
- `processes/renice` - priority control
- Setting rlimits is process-lifetime-scoped in Linux (can't change another live process's hard limits from outside without `prlimit`-style syscalls) - worth confirming feasibility before committing to it

### 10. Disk partitioning (write) — deliberately not recommending yet
`disks/partitions` (read) exists. A write equivalent (`fdisk`/`parted`-style partition table changes) is genuinely one of the most destructive operations a tool like this could expose - a mistake here can be unrecoverable data loss, not just a bad config. If this is ever wanted, it deserves its own dedicated safety design (mandatory dry-run showing the resulting partition table before applying, explicit confirmation token) rather than being added as "just another tool."

## Suggested priority order

1. ✅ **`disks/mounts`** - done.
2. ✅ **`users/list`, `logs/logins`** - done.
3. ✅ **`system://hostname`/`timezone`/`locale`** - done, as resources (read side) - see item 6 above for the categorization correction from the original tool-based plan.
4. **`security/suid-scan`, `security/updates-available`** - next up: read-only, security-relevant, no new architectural questions.
5. **User/group mutation, cron, firewall, package install, `system/hostname|timezone|locale` (set)** - all genuinely useful but all need the dry-run/confirmation governance layer (still open) designed *before* shipping the first one of these, not after - otherwise the same "config exists but doesn't do what's intended" class of bug this session found repeatedly will happen again on tools with real destructive potential.
6. **Container/Kubernetes awareness, disk partitioning (write)** - deliberately last: both are scope decisions (does this daemon's mandate extend to k8s-node-specific tooling? is write-partitioning ever in scope at all?) worth making explicitly rather than by default momentum.
