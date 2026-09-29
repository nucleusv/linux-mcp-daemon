# FR-023 Survey the everyday Linux utilities, map them to mcpd's tools, and propose what to build next (cron first)

- **Created:** 2026-09-29, by the owner ("create task to gather info about regular utils of linux, and suggest what should implement else, I think cron should be ... you need to utilize all utils")
- **Related:** ARCHITECTURE.md (documented exceptions to "no CLI wrapping"), GUIDELINES.md (naming, checklists), `internal/rpc/tools.go` (what exists today), `configs/mcp-sudo.yaml`, `plan/*.md` (stale names - do not trust)

## Description

mcpd's goal is that an agent can do what an admin does at a shell, without SSH, with every action scoped, audited and isolated. Today it covers the read side well (files, disks, processes, network, memory, cpu, services, system, logs, kernel, users) plus docker. The owner's view: coverage should reach the **everyday utilities** an admin reaches for, and cron is an obvious gap. This ticket is the research and the proposal - it builds no tool. Its output is a ranked list of new tools, each small enough to become its own ticket.

Method, kernel-first as the project requires: for each utility, say what it does, which native source could replace it (`/proc`, `/sys`, netlink, DBus, a config file's format, a syscall) or - when nothing native is practical - whether it belongs on the documented-exceptions list like `smartctl` and `traceroute`; what the tool would look like (`<group>/<command>`, read vs. write, parameters); what its security model is (unprivileged, root, grant shape in `mcp-sudo.yaml`, scoping like `paths:`/`containers:`); and whether an agent gains something a file read or an existing tool does not already give.

### Areas to survey (a starting list, to be extended by the research)

- **Scheduling:** cron (`crontab -l/-e`, `/etc/crontab`, `/etc/cron.d`, `cron.{hourly,daily,...}`), `at`/`atq`, **systemd timers** (`systemctl list-timers`, next/last run - already reachable over DBus like `services/*`), anacron. First candidate: `cron/list` (read every user's and system crontabs, parsed) and a scoped `cron/manage` (add/remove one entry for a granted user), plus `timers/list`.
- **Accounts and access:** users/groups (`useradd`, `usermod`, `groupadd`, `passwd`, `id`, `getent`), `sudoers` (read-only view), SSH (`authorized_keys`, `sshd_config` checks, `ssh-keygen -l` fingerprints), `w`/`who`/`last`/`loginctl` sessions.
- **Files:** `mkdir`, `cp`, `mv`, `rm`, `ln`, `touch`, `stat`, `du`/`df` (done), `tar`/`gzip`/`zip`, checksums (`sha256sum`), `diff`, `grep`/`wc`/`sort`/`head`/`tail` (does `files/read` already cover them?), `lsof`/`fuser` (who holds a file), `truncate`.
- **Storage:** `mount`/`umount`/`fstab`, LVM (`pvs/vgs/lvs`), RAID (`/proc/mdstat`), filesystem checks, quotas, swap on/off.
- **Network:** `ip addr/route/link` writes, `ss` (done), `ethtool`, firewall (`nft`/`iptables`/`ufw` - read the ruleset, scoped changes), `/etc/hosts` and `resolv.conf`, `dig` beyond `network/nslookup`, `tc`, wireguard/`wg show`.
- **System:** `hostnamectl`, `timedatectl` / chrony / NTP status, `systemd-analyze`, `journalctl` (done), unit files (read, edit, `daemon-reload`), `needrestart`/pending reboot, `uptime`, `modprobe`/`lsmod` writes, `ulimit`/limits.d, environment.
- **Packages:** install/remove/upgrade (today read-only `system/packages`), `apt`/`dnf` update lists and security advisories, held packages.
- **Security and certs:** `openssl x509 -text` on a cert file, expiry checks, `auditd`/`ausearch`, SELinux/AppArmor status.
- **Containers and virtualization beyond docker:** podman, `kubectl` (probably out of scope), libvirt/`virsh` list.
- **Diagnostics:** `strace`/`perf`/`iostat`/`vmstat`/`sar`, `lsof`, `netstat`, `nmap` (probably out of scope), `smartctl` (done).

## Subtasks

The research is done; what it recommends is built as separate tickets, each one small enough to finish alone. This ticket stays open as the parent until the owner has decided on every item of the ranked list (section (d) under "Research results").

- [FR-025](FR-025-timers-list.md) - `timers/list`: read systemd timers (rank 1) - new
- [FR-026](FR-026-cron-manage.md) - `cron/manage`: read a user's crontab and write it whole, per-user view/edit rules (ranks 2 and 9, cut down by the owner) - new
- Not filed yet (ranks 3-15 in section (d)): the owner picks which become tickets.

## Acceptance criteria

- [ ] An inventory table of the utilities above (and any the research adds): utility, what it does, does mcpd cover it today (which tool), native source available, effort (S/M/L), risk (read / scoped write / root).
- [ ] A ranked proposal of the next tools to build, with cron and systemd timers evaluated first: for each, the tool name, parameters, native approach, grant model in `mcp-sudo.yaml`, and the safety questions to settle (for cron: whose crontab, how an entry is validated, what an agent may schedule, how it is audited, whether it can be undone).
- [ ] Utilities that should stay out, with the reason (needs a TTY, interactive, unbounded output, or too powerful for the scoping model).
- [ ] The owner picks; each accepted item becomes its own backlog ticket with description, criteria and tests.
- [ ] The result is written into this ticket (or a document it links), not left in chat.
- [ ] Definition of Done (backlog/README.md) - for a research ticket: no code, so the docs/build/deploy items are N/A and marked so.

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Baseline is accurate | list every tool and resource from `get mcp-api tools/resources` on a stand and diff against the "covered today" column | no covered tool missing from the table, no listed tool that does not exist | 2026-09-29, repo code (registry read in `internal/rpc/tools.go`, `docker.go`, `resources.go`, `cmd/mcpd/main.go`; no live stand queried) | Pass with findings: 46 tools + 11 resources + 10 templates listed, none missing; 3 stale assumptions found (timers not yet reachable via `services/list`, `auth/sudo-rules` is mcpd's yaml not host sudoers, ARCHITECTURE names `disks/fdisk`/`disks/smartctl`/`network/traceroute` do not exist). Evidence in Research results (f) |
| T2 | Native claims are real | for each "native source" claim, check on a real host (VPS 89.125.210.117, local k8s node) that the file/DBus/netlink source exists and yields the data | each claim backed by real output or marked "needs the CLI" | 2026-09-29, VPS 89.125.210.117 (Ubuntu 24.04, read-only ssh commands only) | Partial pass: cron files, spool permissions, systemd Timer DBus properties, timedate1/hostname1/login1, mdstat, PSI, sysfs net, cgroup v2 verified with output (V1-V15); unverified items listed at the end of (f); local k8s node not checked |
| T3 | Proposal reviewed | the owner reads the ranked list | accepted / rejected / deferred per item, recorded here | | |
| T4 | Follow-ups filed | one ticket per accepted item in `backlog/new/` | each has description, criteria and tests | | |

## Research results

Researched 2026-09-29. Baseline from the code (`internal/rpc/tools.go`, `internal/rpc/docker.go`, `internal/rpc/resources.go`, `cmd/mcpd/main.go`, `internal/tools/*/*/README.md`, `internal/config/sudo.go`, `configs/mcp-sudo.yaml`). Native-source claims checked read-only on the Ubuntu 24.04 VPS (kernel 6.8.0-142). Legend: **[V#]** = verified, evidence E# in section (f); **[U]** = unverified (from general knowledge, no host to test); class **R** = read-only, **SW** = scoped write (grant with `paths:`/`containers:`-style scope), **ROOT** = needs `privileged: true`; effort S/M/L; risk L/M/H.

### (a) Baseline: what mcpd covers today

**46 callable tools** = 38 in `tools.go` (incl. `auth/sudo-rules`, `daemon/reload-config`) + 8 `docker/*` in `docker.go`. Plus internal workers that are not `tools/call` targets (`files/stat`, `files/content`, `files/read`, `processes/read`, `services/status`, `docker/inspect`, `read_usb|pci|dmi|modules|routes|interfaces`). Only `docker/*` and the listed CLI wrappers exec binaries.

| Tool | Purpose | Changes host? |
|---|---|---|
| `files/list` | `ls -la`, native lstat | R |
| `files/read` | file contents | R |
| `files/find` | search by name/type/etc. (**wraps `find`**) | R |
| `files/filetype` | MIME type (native magic bytes) | R |
| `files/create` | create/replace a file (`paths:` scoped) | SW/ROOT |
| `files/update` | append / replace line range (`paths:`) | SW/ROOT |
| `files/chmod`, `files/chown` | mode / owner, symlinks never followed (`paths:`) | SW/ROOT |
| `disks/free`, `disks/usage` | `df` (statfs) / `du -s` (WalkDir) | R |
| `disks/list`, `disks/mounts`, `disks/partitions` | `lsblk` / `mount` / partition bounds from `/sys`, mountinfo, `/proc/swaps` | R |
| `disks/performance` | `iostat` from `/proc/diskstats` | R |
| `disks/health` | SMART (**wraps `smartctl`**, root) | R |
| `processes/list`, `processes/top` | `ps` / `top -b -n1` from `/proc` | R |
| `processes/delete` | send a signal (kill) | **writes** |
| `memory/usage`, `cpu/list`, `cpu/load-average` | `/proc/meminfo`, `/proc/cpuinfo`, sysinfo | R |
| `network/connections` | `ss -tuanp` from `/proc/net/*` | R |
| `network/arp`, `network/nslookup`, `network/ping` (TCP), `network/curl` | ARP cache, native DNS, TCP reach, HTTP client | R (curl can POST: outbound only) |
| `network/trace-path` | **wraps `traceroute`** | R |
| `services/list` | systemd units over DBus - **only `*.service`** (list.go filters the suffix) | R |
| `services/manage` | start/stop/restart/reload/enable/disable a unit over DBus | **writes** |
| `logs/journal-control` (**wraps `journalctl`**), `logs/dmesg` (**wraps `dmesg`**), `logs/logins` (**wraps `last`/`lastb`**) | logs | R (journal-control has control actions - see its README) |
| `kernel/system-control` | sysctl via `/proc/sys`, write keys scoped by `sysctl.write_keys` | SW/ROOT |
| `system/os-release`, `system/packages` | uname + os-release; dpkg/apk db (no rpm) | R |
| `users/list` | `/etc/passwd` + `/etc/group` | R |
| `auth/sudo-rules` | **mcpd's own** `mcp-sudo.yaml` grants for the caller - NOT the host's sudoers | R |
| `daemon/reload-config` | re-read mcpd configs | mcpd config |
| `docker/containers`, `images`, `volumes`, `networks`, `logs` | Engine API listings / logs (root socket, `containers:` scoped) | R |
| `docker/manage`, `docker/exec`, `docker/prune` | lifecycle / run in container / prune (`containers:`, `prune:` scoped) | **writes** |

**Resources (static, 11):** `os://uname`, `os://release`, `system://hostname`, `system://timezone`, `system://locale`, `network://interfaces`, `network://routes`, `devices://usb|pci|dmi` (root), `kernel://modules` (root, = `lsmod`). **Templates (10):** `file:///{path}` (+ `/stat`, `/content`, `/type`), `devices://{type}`, `network://interfaces/{name}`, `service://{name}/status`, `disks://{name}/stats`, `container://{name}/{view}`, `image://{name}/inspect`, `volume://{name}/inspect`, `docker-network://{name}/inspect`, `process://{pid}/{target}` (status, cmdline, environ, limits, open_files). All read-only.

Existing infrastructure to reuse: `logging.Audit(...)` (every host-changing call is audited, `internal/rpc/tools.go:985`); per-tool grant keys `paths:`, `containers:`, `prune:`, `sysctl.write_keys` validated strictly in `internal/config/sudo.go` ("allowed without scope = refuse"); `go-systemd/v22` already a dependency; documented CLI exceptions currently in code: `traceroute`, `dmesg`, `last`/`lastb`, `journalctl`, `smartctl`, `find`.

### (b) Inventory of everyday utilities (83 rows)

Coverage: **Full** / **Part** / **None**. "CLI" = no practical native route.

| # | Utility | What it does | mcpd today | Native source / CLI | Proposed tool | Class | Eff | Risk |
|---|---|---|---|---|---|---|---|---|
| **Scheduling** |||||||||
| 1 | `crontab -l`, `/etc/crontab`, `/etc/cron.d` | list scheduled jobs | Part: `files/read` gives raw text only, no parse, spool not readable unprivileged [V6] | parse files (see section c) | `cron/list` | R (ROOT for user spool) | M | L |
| 2 | `crontab -e/-r` | add/remove jobs | None | write file, validate 5-field schedule | `cron/manage` | SW+ROOT | M-L | H |
| 3 | `cron.{hourly,daily,weekly,monthly}` (run-parts) | periodic scripts | Part (`files/list`) | dir listing + mode bits [V4] | in `cron/list` | R | S | L |
| 4 | `anacron` | run missed periodic jobs | None (not installed on VPS [V8]) | `/etc/anacrontab`, `/var/spool/anacron/*` [U] | in `cron/list` | R | S | L |
| 5 | `systemctl list-timers` | timers, next/last run | **None** (`services/list` filters `.service`) | DBus `ListUnitsByPatterns("*.timer")` + `org.freedesktop.systemd1.Timer` props [V7] | `timers/list` | R | S | L |
| 6 | `systemd-run --on-calendar` / `at` | one-shot delayed run | None | DBus `Manager.StartTransientUnit` exists [V7]; `at` not installed [V8] | `timers/run` | SW+ROOT | M | H |
| 7 | `at`/`atq`/`atrm` | one-off jobs | None | spool `/var/spool/cron/atjobs` (Debian) - dir absent on VPS [V8] | `at/list` (only if `at` is common) | R | S | L |
| **Accounts & access** |||||||||
| 8 | `getent passwd/group`, `id`, `groups` | identity | Full: `users/list` | `/etc/passwd`, `/etc/group` [V13] | - | R | - | - |
| 9 | `useradd/usermod/userdel/groupadd/chage` | change accounts | None | write passwd/shadow/group with locking, or CLI | `users/manage` (defer) | ROOT | L | H |
| 10 | `passwd`, `chpasswd` | set password | None | interactive / secrets | keep out | - | - | - |
| 11 | `sudo -l`, `/etc/sudoers`, `sudoers.d` | who may run what | **None** (`auth/sudo-rules` is mcpd's yaml); file is 0440 root [V9] | parse file + includes | `users/sudoers` | R ROOT | M | L |
| 12 | `w`, `who`, `loginctl list-sessions` | live sessions | None (`logins` = history) | DBus `login1.Manager.ListSessions` [V10] or `/run/utmp` | `users/sessions` | R | S | L |
| 13 | `last`, `lastb` | login history | Full: `logs/logins` (wraps CLI) | `/var/log/wtmp`, `btmp` binary utmp records (btmp root, 141 MB on VPS [V14]) | native utmp parser (optional) | R | M | L |
| 14 | SSH `authorized_keys` | who can log in | Part (`files/read` shows keys, no fingerprints) | parse + SHA256 fingerprint of key blob (stdlib) [V9] | `ssh/keys` | R (ROOT other homes) | S | L |
| 15 | `sshd -T`, `sshd_config` | effective sshd policy (PermitRootLogin, ports) | Part (`files/read`) | parse config + `Include`/`sshd_config.d` [V9]; exact `Match` semantics need `sshd -T` | `ssh/config` | R | M | L |
| 16 | `ssh-keygen -l`, host keys | fingerprints | None | parse `/etc/ssh/*.pub` | in `ssh/keys` | R | S | L |
| 17 | `faillock`/`pam_tally2`, PAM config | lockout state | None | `/var/run/faillock`, `/etc/pam.d` [U] | defer | R | M | L |
| 18 | `ulimit`, `limits.conf`, `limits.d` | resource limits | Part: `process://{pid}/limits` per PID; conf files by `files/read` [V11] | parse `/etc/security/limits*` | `system/limits` | R | S | L |
| 19 | `env`, `/etc/environment`, `/etc/default/*` | environment | Part (`files/read`); `process://pid/environ` | file parse | not needed | R | - | - |
| **Files** |||||||||
| 20 | `ls`, `stat`, `file`, `find`, `du`, `df` | inspect | Full | syscalls | - | R | - | - |
| 21 | `cat`, `head`, `tail`, `wc`, `grep`, `sort` | read/filter | Part: `files/read` (offset/limit per its schema; no grep/tail) | stdlib | `files/read` options or `files/grep` | R | S | L (unbounded output: cap) |
| 22 | `mkdir`, `touch`, `ln -s` | create | None (files/create writes a file; dir creation not confirmed in code) | `mkdir(2)`, `symlink(2)` | `files/mkdir` | SW | S | M |
| 23 | `cp`, `mv` | copy / move | None | `io.Copy` / `rename(2)` | `files/copy`, `files/move` | SW | M | M |
| 24 | `rm`, `rmdir`, `truncate` | delete | None | `unlink(2)`; no recursion by default | `files/delete` | SW | M | H |
| 25 | `chmod`, `chown` | perms | Full | syscalls | - | SW | - | - |
| 26 | `tar`, `gzip`, `unzip` | archives | None | `archive/tar`, `compress/gzip`, `archive/zip` (stdlib); zip-slip must be handled | `files/archive` | SW | M | M |
| 27 | `sha256sum`, `md5sum` | checksums | None | `crypto/sha256` | `files/checksum` | R | S | L |
| 28 | `diff` | compare | None | stdlib-only diff is L; low value (agent diffs in its head) | skip | R | M | L |
| 29 | `lsof`, `fuser` | who holds a file/port | Part: `process://{pid}/open_files` forward only; sockets via `network/connections` | scan `/proc/*/fd` (reverse lookup) | `files/holders` (an empty `internal/tools/files/get_open_files/` dir exists, unwired) | R ROOT | M | L |
| 30 | `rsync`, `scp` | sync / remote copy | None | needs CLI, remote exfil | keep out | - | - | - |
| **Storage** |||||||||
| 31 | `mount`, `findmnt`, `lsblk`, `fdisk -l` | list | Full | `/proc/mounts`, `/sys/class/block` | - | R | - | - |
| 32 | `mount`/`umount` (write) | attach fs | None | `mount(2)` syscall; scope by source+target | `disks/mount` (defer) | ROOT | M | H |
| 33 | `/etc/fstab` | boot mounts | Part (`files/read`) [V13] | parse | in `disks/mounts` option | R | S | L |
| 34 | `swapon/swapoff`, `/proc/swaps` | swap | Part: `disks/list` reads swaps [V11] | `swapon(2)` | - | ROOT | S | M |
| 35 | `cat /proc/mdstat`, `mdadm --detail` | software RAID health | None | `/proc/mdstat` [V11] | `disks/raid` | R | S | L |
| 36 | LVM `pvs/vgs/lvs` | logical volumes | None | `/sys/block/dm-*` covers active LVs only; metadata needs `lvs --reportformat json` CLI [V11: no dm-* on VPS] | `disks/lvm` (CLI exception) | R ROOT | M | L |
| 37 | `fsck`, `tune2fs`, `mkfs`, `dd`, `parted` | filesystem work | None | destructive | keep out | - | - | - |
| 38 | `quota`, `repquota` | disk quota | None | `quotactl(2)` [U] | defer | R | M | L |
| 39 | SMART | drive health | Full: `disks/health` (CLI exception) | - | - | R | - | - |
| **Network** |||||||||
| 40 | `ip addr/link/route` (read) | interfaces, routes | Full: `network://interfaces`, `network://routes` (IPv4 only, `/proc/net/route` [V12]) | `/sys/class/net`, netlink | IPv6 routes: extend resource | R | S | L |
| 41 | `ip ... add/del` | change addr/routes | None | netlink (needs `x/sys` or hand-rolled) | keep out (lockout risk) | ROOT | L | H |
| 42 | `ss`, `netstat` | sockets | Full: `network/connections` | `/proc/net/*` | - | R | - | - |
| 43 | `ethtool` | link speed/duplex/driver | None | `/sys/class/net/<if>/{speed,duplex,operstate,mtu}` [V12]; offloads need ioctl | `network/link` | R | S | L |
| 44 | `nft list ruleset`, `iptables -S`, `ufw status` | firewall rules | None | nft via netlink needs a library (zero-dep rule); `/proc/net/ip_tables_names` was unreadable on VPS; `nft`, `iptables`, `ufw` binaries present [V12] -> CLI exception `nft -j list ruleset` | `network/firewall` | R ROOT | M | L |
| 45 | firewall writes | open/close ports | None | CLI | keep out (lockout, RCE-equivalent) | ROOT | L | H |
| 46 | `/etc/hosts`, `resolv.conf` | name resolution config | Part (`files/read`); `network/nslookup` uses Go resolver [V12] | file parse | not needed | R | - | - |
| 47 | `dig` | full DNS | Part (`nslookup`: types A/etc.) | Go `net` / custom resolver | extend `nslookup` (MX/TXT/SRV, @server) if missing | R | S | L |
| 48 | `conntrack -L`, `/proc/net/nf_conntrack` | NAT table | None | absent on VPS (module not loaded) [V12] | defer | R | S | L |
| 49 | `/proc/net/dev`, `nstat`, `ip -s` | counters | Full (`network://interfaces` RX/TX) | - | - | R | - | - |
| 50 | `tc`, `wg show`, `bridge` | shaping, wireguard | None | netlink / CLI | defer | R | L | L |
| 51 | `nmap`, `tcpdump` | scan / capture | None | - | keep out | - | - | - |
| **System** |||||||||
| 52 | `hostnamectl`, `timedatectl status` | hostname, tz, NTP | Part: `system://hostname`, `system://timezone`; sync state missing | DBus `hostname1`, `timedate1` props `NTP`, `NTPSynchronized`, `Timezone` [V10] | `system/time` | R | S | L |
| 53 | chrony/timesyncd status | clock sync | None | `timedatectl show-timesync` = DBus `timesync1`; `chronyc` if chrony [V10] | in `system/time` | R | S | L |
| 54 | `uptime`, `/proc/uptime`, PSI | uptime, pressure | Part: `cpu/load-average`; `/proc/pressure/*` not exposed [V11] | `/proc/uptime`, `/proc/pressure/*` | `system/pressure` (or extend load-average) | R | S | L |
| 55 | `needrestart`, `/var/run/reboot-required` | reboot pending | None | file + `.pkgs` list [V11: absent = no reboot needed] | `system/reboot-required` | R | S | L |
| 56 | `systemd-analyze`, `blame` | boot time | None | DBus Manager props `*TimestampMonotonic`; blame needs per-unit props | defer | R | M | L |
| 57 | failed units (`systemctl --failed`) | health | Part: `services/list active_state=failed` [V13: 2 failed units on VPS] | DBus | - | R | - | - |
| 58 | unit files: `systemctl cat`, edit, `daemon-reload` | inspect/override units | Part: `files/read/update` on unit files, but **no daemon-reload** tool | DBus `Manager.Reload`, `GetUnitFileState`, `ListUnitFiles` [V13] | `services/reload` or `services/list-files` | R / SW | S | M |
| 59 | `modprobe`, `insmod`, `rmmod` | load/unload modules | None (`kernel://modules` read only) | `init_module`/`delete_module` syscalls; scope by module name allowlist | `kernel/modules` (defer) | ROOT | M | H |
| 60 | `/etc/modprobe.d`, blacklists | module config | Part (`files/read`) [V11] | file | not needed | R | - | - |
| 61 | `sysctl` | kernel params | Full | - | - | SW | - | - |
| 62 | `lsmod`, `lspci`, `lsusb`, `dmidecode` | hardware | Full: `kernel://modules`, `devices://*` | - | - | R | - | - |
| 63 | `sensors`, thermal | temperatures | None; VPS has no hwmon [V12] | `/sys/class/hwmon`, `/sys/class/thermal` | `system/sensors` (bare-metal only) | R | S | L |
| 64 | `reboot`, `shutdown`, `poweroff` | power | None | DBus `logind.Reboot` | keep out until an approval/confirm model exists | ROOT | S | H |
| 65 | `logrotate` state/config | rotation | Part (`files/read`); config lives `/etc/logrotate.d`, timer `logrotate.timer` [V4,V7] | parse `/etc/logrotate.conf`, `/var/lib/logrotate/status` | `logs/rotation` | R | S | L |
| 66 | journal disk usage/vacuum | journal size | Part: `logs/journal-control` (see README) [V14: 1.5G] | `/var/log/journal`, DBus/`journalctl --disk-usage` | check `journal-control` scope | R/SW | S | L |
| 67 | `dmesg`, `journalctl` | logs | Full (CLI exceptions) | - | - | R | - | - |
| 68 | cgroups / resource control (`systemd-cgtop`, `MemoryMax`) | per-service usage | None | `/sys/fs/cgroup/**` (v2 [V11: `cgroup.controllers` present]) | `system/cgroups` | R | M | L |
| **Packages** |||||||||
| 69 | `dpkg -l`, `apk info` | installed | Full (no rpm) | dpkg status | rpm via sqlite = out of native scope | R | - | - |
| 70 | `apt list --upgradable`, `dnf check-update` | pending updates / security advisories | None | apt lists are compressed indexes -> CLI `apt-get -s upgrade` or `apt list --upgradable` | `system/updates` (CLI exception) | R ROOT | M | L |
| 71 | held packages, pins | `apt-mark showhold` | None | dpkg status `Status: hold ...` + `/etc/apt/preferences.d` [V11] | in `system/packages` filter | R | S | L |
| 72 | `apt install/remove/upgrade`, `dnf` | change packages | None | CLI, long-running, scripts as root | `packages/manage` (defer) | ROOT | L | H |
| 73 | `unattended-upgrades` status | auto-update health | None | `/var/lib/apt/periodic/*-stamp` [V11], `/var/log/unattended-upgrades` | in `system/updates` | R | S | L |
| **Security & certs** |||||||||
| 74 | `openssl x509 -text`, expiry | inspect certs, days left | None (Part: `files/read` of PEM) | `crypto/x509` + `encoding/pem` (stdlib); live check via `crypto/tls` | `security/certs` | R | S-M | L |
| 75 | trust store, letsencrypt | CA bundle, `/etc/letsencrypt/live` | None; 243 CA entries, no letsencrypt on VPS [V15] | scan dirs | in `security/certs` | R | S | L |
| 76 | AppArmor / SELinux status | MAC state | None | `/sys/kernel/security/lsm` [V11], `/sys/kernel/security/apparmor/profiles`, `/sys/fs/selinux/enforce` [U for selinux] | `security/lsm` | R (ROOT profiles) | S | L |
| 77 | `auditctl`, `ausearch` | audit records | None; binaries absent on VPS [V11] | netlink audit / logs | defer | R ROOT | L | L |
| 78 | `fail2ban-client`, ban lists | intrusion state | None | `fail2ban` socket/CLI [U] | defer | R | M | L |
| **Containers & virt** |||||||||
| 79 | `podman`, `lxc`, `virsh` | other runtimes | None; VPS has `lxc` binary only, no podman/virsh [V12] | podman: Docker-compatible API socket (reuse `internal/docker` client) | `podman/*` only if requested | R/SW | M | M |
| 80 | `kubectl` | k8s | None | own API, huge scope | keep out | - | - | - |
| **Diagnostics** |||||||||
| 81 | `vmstat`, `iostat`, `sar`, `pidstat` | perf snapshots | Part: `memory/usage`, `disks/performance`; no `/proc/vmstat`, `/proc/stat` deltas | `/proc/vmstat`, `/proc/stat`, sysstat files (VPS runs sysstat) | `system/vmstat` | R | S | L |
| 82 | `strace`, `perf`, `gdb`, `lsof -p` | tracing | None | ptrace = code-exec level | keep out | - | - | - |
| 83 | `renice`, `ionice`, `prlimit`, `chrt` | process priority/limits | None | `setpriority(2)`, `prlimit(2)` (native) | `processes/renice` (scoped: only lower priority) | SW | S | M |

### (c) Deep-dive: cron, systemd timers, at

#### c1. Cron: where jobs live (facts from the Ubuntu 24.04 VPS, [V1-V6])

| Source | Path | Format | Readable unprivileged? |
|---|---|---|---|
| System crontab | `/etc/crontab` | 5 fields + **user** + command; `NAME=value` env lines | yes (0644) |
| Drop-ins | `/etc/cron.d/*` | same as `/etc/crontab`. Debian ignores names with a dot (the `.placeholder` file on the VPS) and requires root-owned, non-group/world-writable files [U for the exact rule] | yes (0644) |
| Periodic scripts | `/etc/cron.{hourly,daily,weekly,monthly}/*` | executable scripts run by `run-parts`; invoked by lines in `/etc/crontab` (guarded by `test -x /usr/sbin/anacron \|\|`) or by anacron | yes (dir listing) |
| Per-user crontabs | Debian: `/var/spool/cron/crontabs/<user>`; RHEL/cronie: `/var/spool/cron/<user>` [U] | 5 fields + command, **no user field** (the file name is the user) | **no**: dir is `drwx-wx--T root:crontab`, `nobody` gets "Permission denied" [V6]. Reading needs root (or group `crontab`) |
| Access control | `/etc/cron.allow`, `/etc/cron.deny` | one user per line; allow beats deny; neither present on VPS [V3] (Debian then allows everyone; RHEL semantics [U]) | yes |
| Anacron | `/etc/anacrontab`, `/var/spool/anacron/<job>` | `period delay id command` | not installed on VPS [V8] |

Line grammar to implement (Vixie/Debian cron; [U] details from cron(5), not run on host): five fields `min hour dom month dow`, each `*`, number, `a-b`, `a,b,c`, `*/n`, `a-b/n`; month and weekday names (`jan`, `mon`) allowed as single values; dow 0 and 7 = Sunday; if both dom and dow are restricted the job runs when **either** matches; macros `@reboot @yearly @annually @monthly @weekly @daily @midnight @hourly`; env lines `NAME=value` (`SHELL`, `PATH`, `MAILTO`, `CRON_TZ`, `HOME`), no variable expansion, apply to the lines after them in that file; `%` in the command is a newline/stdin separator unless escaped `\%`; system crontabs put a username between schedule and command (seen live: `17 * * * * root cd / && run-parts --report /etc/cron.hourly`, and a non-stock `@reboot root /opt/scan.sh` [V2]). Comments start at line start only. Debian cron drops a whole crontab on a syntax error [U] - reason enough to validate before writing.

Live examples that the parser must survive [V2]: tabs mixed with spaces as separators; ranges with steps `5-55/10`; commands with `||`, `&`, `{ ...; }`; `@reboot` in `/etc/crontab`; a `PATH=` line before jobs; guards like `[ -d /run/systemd/system ] ||` (a job that is a no-op under systemd - `next_run` for these would be misleading, so `cron/list` must not claim the job will do work, only when cron will invoke it).

#### c2. `cron/list` - read natively, no `crontab` binary

Parse the files above in the worker (never the master). Unprivileged run covers `/etc/crontab`, `/etc/cron.d`, periodic dirs, allow/deny; `privileged: true` adds `/var/spool/cron/crontabs/*` (or `/var/spool/cron/*`). Detect the flavour by which spool dir exists. Cap: per-entry command length, total entries; never execute anything. Redact obvious secrets in commands (existing `internal/rpc/redact_test.go` suggests a redaction helper exists - check reuse; open question 9).

```json
{ "name": "cron/list", "tools_group": "cron", "linuxctl_verb": "cron",
  "inputSchema": { "type": "object", "properties": {
    "user":       {"type": "string", "description": "Only jobs that run as this user"},
    "source":     {"type": "string", "enum": ["all","system","user","drop-in","periodic"], "description": "default all"},
    "next_runs":  {"type": "integer", "description": "Compute the next N fire times per job (0-5, default 0)"},
    "privileged": {"type": "boolean", "description": "Also read per-user spool crontabs (root-only)"},
    "output_format": {"type": "string"} } } }
```
Output (JSON):
```json
{ "flavour": "debian", "cron_running": true,
  "access": {"allow_file": false, "deny_file": false},
  "entries": [ {"id": "c-3fa91c07", "source": "/etc/cron.d/atop", "kind": "drop-in",
      "line": 4, "run_as": "root", "schedule": "0 0 * * *", "macro": null,
      "command": "[ -d \"/run/systemd/system\" ] || /usr/share/atop/atop.daily&",
      "env": {"PATH": "/bin:/usr/bin:/sbin:/usr/sbin"}, "managed_by": null, "next_runs": [] } ],
  "periodic": [ {"dir": "/etc/cron.daily", "runner": "cron", "scripts": [{"name": "logrotate", "mode": "0755", "runs": true}]} ],
  "parse_errors": [ {"source": "/etc/cron.d/x", "line": 3, "error": "bad hour"} ] }
```
`id` = `c-` + first 8 hex of sha256(source + normalized schedule + run_as + command) - stable while the line is unchanged; for mcpd-created jobs the id is the one embedded in the job's marker (see c3). `runs:false` for non-executable or dot/`~`/`.dpkg-*` files that run-parts skips. `cron_running` from `/proc` comm scan or DBus `cron.service` state.

#### c3. `cron/manage` - safe write design

**Recommended v1 shape: mcpd only writes files it owns under `/etc/cron.d/`, one job per file `mcpd-<id>` (0644 root:root), never edits a person's crontab.** Why: (1) a root worker can write it with one atomic `rename(2)`; (2) no need to parse/rewrite hand-edited user crontabs (round-tripping comments and env lines is the classic source of clobbered crontabs); (3) removal is deleting one file and undo is restoring one file; (4) the user field is explicit, so "whose crontab" becomes "which `run_as`", which is grantable. Per-user spool editing (`crontab -u`) is a v2 with a marker-delimited managed block, only if the owner needs it (spool files must be `user:crontab 0600`; Debian cron re-reads by mtime [U]).

File content written (marker line makes ownership provable and the entry auditable):
```
# mcpd:managed id=9f2a1b7c by=alice at=2026-09-29T08:12:00Z reason="nightly backup"
MAILTO=""
15 2 * * * backup /usr/local/bin/backup.sh --nightly
```
Write path in the worker: validate -> render -> write temp file in `/etc/cron.d/` (same fs) with `O_EXCL`, fsync -> `rename` -> audit. The master only routes the call; the worker does everything (project rule).

```json
{ "name": "cron/manage", "tools_group": "cron", "linuxctl_verb": "cron",
  "inputSchema": { "type": "object", "properties": {
    "action":   {"type": "string", "enum": ["add","remove","disable","enable"]},
    "id":       {"type": "string", "description": "remove/disable/enable: the id from cron/list (only mcpd-managed jobs)"},
    "schedule": {"type": "string", "description": "add: five fields or one macro (@daily...); validated, normalized in the reply"},
    "run_as":   {"type": "string", "description": "add: the user the job runs as (must be in the grant's run_as list)"},
    "command":  {"type": "string", "description": "add: absolute path of the program plus arguments; no shell operators unless the grant allows shell"},
    "reason":   {"type": "string", "description": "add: stored in the marker line and the audit record"},
    "dry_run":  {"type": "boolean", "description": "validate and show the resulting file and next 3 run times, write nothing"},
    "privileged": {"type": "boolean", "description": "required: the file is root-owned"} },
    "required": ["action"] } }
```
Output: `{ "id": "9f2a1b7c", "file": "/etc/cron.d/mcpd-9f2a1b7c", "line": "15 2 * * * backup /usr/local/bin/backup.sh --nightly", "schedule_normalized": "15 2 * * *", "next_runs": ["2026-09-30T02:15:00Z", "..."], "previous": null, "audit": "logged" }`. `remove` returns the removed file text (the undo record); re-adding it is the undo. `disable`/`enable` toggle by renaming to `mcpd-<id>.disabled` (dot-suffix names are ignored by cron: [U]) - cleaner than commenting out.

Validation (all in Go, unit-testable without a host): schedule parser implementing the grammar in c1 with min/max per field, macros; command must be non-empty, no NUL/newline, no unescaped `%`, first token an absolute path that exists and is executable, resolved with `EvalSymlinks` and required to fall under the grant's `paths:`; shell metacharacters (`; & | < > $ backtick ( ) { }`) refused unless the grant says `shell: true`; `run_as` must exist in `/etc/passwd` and be in the grant; `@reboot` needs an explicit grant; minimum interval computed by expanding minute/hour fields (default refuse more often than every 5 minutes); max managed jobs per user (default 20); the job may only set `MAILTO`/`CRON_TZ` (default `MAILTO=""` to avoid mail floods - see question 8), never `SHELL`/`PATH`/`LD_*`.

Grant shape, modelled on `paths:`/`containers:` (absent scope = refuse everything, misspelled key = load error, same strictness as `internal/config/sudo.go`):
```yaml
cron/manage:
  allowed: true
  paths:            # the executable must live under one of these
    - /usr/local/bin
    - /opt/myapp/bin
  cron:
    run_as: [backup, www-data]     # explicit; "root" only if listed
    min_interval: 15m              # default 5m
    max_jobs: 20                   # mcpd-managed jobs in total for this user
    shell: false                   # true allows pipes/&& in command (much wider)
    allow_reboot: false            # @reboot
```
`cron/list` needs only `allowed: true` (plus `privileged` for the spool). New keys must be added to `sudo.go`'s validation and to `configs/mcp-sudo.yaml`'s `privileged` block and docs per CLAUDE.md steps 1-10.

Audit: reuse `logging.Audit("tool call", ...)` and add fields `cron_id`, `file`, `run_as`, `schedule`, sha256 of the file, and the full line for `add` (commands can contain secrets - see question 9). Also keep an append-only `/var/lib/mcpd/cron-undo/<id>.<ts>` copy of removed files so undo survives (open question 6).

#### c4. systemd timers (`timers/list`) - verified path

Read over DBus exactly like `services/list`: `conn.ListUnitsByPatternsContext(ctx, nil, []string{"*.timer"})` (go-systemd already imported; busctl call on the VPS returned 18 timers [V7]) then per unit `GetUnitTypePropertiesContext(name, "Timer")` for the `org.freedesktop.systemd1.Timer` properties (verified names: `TimersCalendar`, `TimersMonotonic`, `NextElapseUSecRealtime`, `NextElapseUSecMonotonic`, `LastTriggerUSec`, `Unit`, `Persistent`, `RandomizedDelayUSec`, `Result` [V7]). Readable by an unprivileged user (the system bus allows reads; `nobody` got the property [V7]), so no grant scope needed beyond `allowed: true`. `services/list`'s `.service` filter means a timer-only change is a new tool, not a flag on the old one; alternatively add `type: service|timer` to `services/list` - owner decision (question 13).
```json
{ "name": "timers/list", "tools_group": "system", "linuxctl_verb": "timers",
  "inputSchema": { "type": "object", "properties": {
    "pattern":       {"type": "string", "description": "glob on the timer name, e.g. 'apt-*'"},
    "active_state":  {"type": "string", "description": "active | inactive | failed"},
    "include_inactive": {"type": "boolean", "description": "default true (systemctl list-timers --all)"},
    "output_format": {"type": "string"} } } }
```
Output: `{"timers":[{"unit":"apt-daily.timer","activates":"apt-daily.service","active_state":"active","sub_state":"waiting","on_calendar":["*-*-* 06,18:00:00"],"on_monotonic":[],"next":"2026-09-29T10:12:15Z","last":"2026-09-28T20:40:39Z","persistent":true,"randomized_delay_sec":43200,"last_result":"success"}]}` (unit, schedule, next/last and persistent values are from [V7]; `randomized_delay_sec` and `last_result` are illustrative, not captured). Note [V7]: `systemctl list-timers` shows a timer with no schedule (`-`) when inactive/never armed (`apport-autoreport.timer`). Write side later: **transient timers** (`Manager.StartTransientUnit` with `OnCalendar`/`OnActiveSec` and an `ExecStart` via `systemd-run` semantics) give one-shot "run this in 10 minutes" without touching any file and vanish on reboot - a safer replacement for `at` than a new spool writer; persistent timers need unit-file writes plus `daemon-reload` (`files/create` + `Manager.Reload` exist [V13]) and are a separate, riskier tool. User-scope timers (`systemctl --user`) need the per-user bus: out of v1.

#### c5. `at` / `atq`

`at` is not installed on the VPS and its spool is absent [V8]; Ubuntu 24.04 does not ship it by default. Spool (from general knowledge [U]): Debian `/var/spool/cron/atjobs/<a+5hex+8hex>` (job files are shell scripts with `# atrun uid=.. gid=..` header, environment dump and the command), RHEL `/var/spool/at`. Recommendation: **no `at` tool**; read-only `at/list` only if a target fleet uses it; delayed execution via transient timers (c4).

#### c6. Open safety questions the owner must decide

1. **Whose crontab**: OK to limit v1 to mcpd-owned files in `/etc/cron.d` (recommended), or is editing real per-user crontabs (`crontab -u`) required?
2. **What may an agent schedule**: only executables under granted `paths:` and `run_as` users (recommended), or arbitrary shell lines? Is `root` ever a permitted `run_as`? (Scheduling a command as root that the agent may not run directly is a privilege-escalation channel that outlives the session; `privileged:true` for `cron/manage` plus `run_as: root` should probably be its own explicit line in the grant.)
3. **Shell**: allow `shell: true` (pipes, `&&`, redirects), which makes the `paths:` scope meaningless? Default off?
4. **Frequency and count limits**: minimum interval (proposed 5-15 min) and max jobs (proposed 20) - agree numbers; are `@reboot` jobs allowed? (they are persistence.)
5. **Update vs add-only**: allow editing an existing job, or only add/remove/disable (proposed: no in-place edit)?
6. **Undo**: is "remove returns the old text" enough, or should mcpd keep an undo store under `/var/lib/mcpd/`? Who can restore?
7. **Foreign jobs**: may `cron/manage` remove/disable jobs it did not create (proposed: never)? May `cron/list` show them (yes, read-only)?
8. **Mail**: cron mails job output to `MAILTO`/the user; default `MAILTO=""`? Where should output go (agent cannot see it unless redirected)? Should mcpd offer a log-file argument scoped by `paths:`?
9. **Secrets**: commands may embed tokens or passwords; redact in `cron/list` and audit (which rule?), or forbid inline credentials by validation?
10. **Audit sink**: is the existing `logging.Audit` line enough, or do we also want syslog/journal entries tagged `mcpd-cron`? Retention of the removed-job text?
11. **Time zone**: cron uses the daemon's local TZ (or `CRON_TZ`); is `next_runs` reported in UTC only? (DST gaps: 02:30 does not exist once a year.)
12. **Containerised mcpd** (`worker.containerized`): the host's `/etc/cron.d` is reached through `privileged: true` + host mount namespace like `files/*`; confirm that is acceptable, and remember the ARCHITECTURE note that `privileged` conflates root and host-namespace.
13. **Timers**: separate `timers/list` (proposed) or a `type` filter on `services/list`? Persistent timer creation (unit files + reload) in scope now or later?
14. **Non-Debian**: RHEL spool path and cronie's differences are untested here; support Debian-family first?
15. **Observation, not a question**: `/etc/crontab` on the VPS contains a non-stock line `@reboot root /opt/scan.sh` (root-owned 230-byte script, mtime 2024-06-27 [V2]); it is the kind of persistence entry `cron/list` should make visible. Contents were deliberately not read; the owner may want to confirm it is theirs.

### (d) Ranked proposal: next tools to build

Score = value to an agent / effort / risk. Everything ranked 1-8 is read-only.

| Rank | Tool | Why | Native approach | Grant | Eff | Risk |
|---|---|---|---|---|---|---|
| 1 | `timers/list` | closes the "what runs on a schedule" half of every incident; on the VPS the *only* real scheduler for ~18 jobs [V7]; smallest change (same DBus code as `services/list`) | DBus `Timer` properties [V7] | `allowed: true` | S | L |
| 2 | `cron/list` | owner's first candidate; jobs invisible today unless root reads the spool by hand [V6]; makes persistence visible [V2] | file parse (c2) | `allowed: true`; root for user spool | M | L |
| 3 | `ssh/keys` + `ssh/config` (one ticket, two tools) | who can log in and is root/password login allowed (VPS: `PermitRootLogin yes` [V9]); pure parsing, stdlib fingerprints | parse `authorized_keys`, `sshd_config`+`.d` | `allowed: true`, root for other homes | S-M | L |
| 4 | `security/certs` | expiry checks are the most common admin outage; stdlib `crypto/x509` + `crypto/tls`; no CLI | PEM parse, TLS dial | `allowed: true` (+ `paths:` for cert dirs) | S-M | L |
| 5 | `system/time` (+ pending-reboot in `system/reboot-required`) | clock skew breaks TLS/cron/kerberos; DBus `timedate1` verified [V10]; reboot flag is a file [V11] | DBus, file | `allowed: true` | S | L |
| 6 | `users/sessions` (`who`/`w`) | who is on the box now; complements `logs/logins`; DBus `login1` verified [V10] | DBus `ListSessions` | `allowed: true` | S | L |
| 7 | `network/firewall` (read) | the top question when a port "does not work"; CLI exception (`nft -j`) like `smartctl` | `nft -j list ruleset`, fallback `iptables -S` [V12] | `allowed: true` + root | M | L |
| 8 | `disks/raid` (+ `disks/lvm` later) | health of storage; `/proc/mdstat` is a plain file [V11] | `/proc/mdstat`; LVM via CLI exception | `allowed: true` | S / M | L |
| 9 | `cron/manage` (`/etc/cron.d/mcpd-*` model) | the write half the owner wants; ranked below the reads only because several of the questions in (c6) need answers first | c3 | `paths:` + new `cron:` block | M-L | H |
| 10 | `files/mkdir`, `files/copy`, `files/move`, `files/delete` (one ticket, four tools) | the largest functional hole: an agent can write and chmod but not create a directory, copy, move or remove; scoped by existing `paths:` | syscalls / stdlib | `paths:` | M | M (delete: H, non-recursive default) |
| 11 | `users/sudoers` (host sudoers view) | today `auth/sudo-rules` is mcpd's yaml, not the host's, and the `privileged` block comment claims otherwise; answers "who has root here" | parse `/etc/sudoers` + `sudoers.d`, root [V9] | `allowed: true` + root | M | L |
| 12 | `timers/run` (transient timer, one-shot `at` replacement) | delayed one-off jobs without a file or spool | DBus `StartTransientUnit` [V7] | new scoped grant like cron's | M | H |
| 13 | `system/updates` (upgradable + held + auto-update stamps) | patch hygiene; needs `apt` CLI exception [V11] | CLI `apt-get -s`/`apt list --upgradable` | root | M | L |
| 14 | `system/pressure`/`vmstat` (PSI, `/proc/vmstat`) | fast "is the box starving" signal; PSI present [V11] | procfs | `allowed: true` | S | L |
| 15 | `processes/renice` | scoped priority change instead of `kill` | `setpriority(2)` | scoped (only positive nice) | S | M |

Also cheap and worth a ticket each if wanted: `network/link` (ethtool basics from sysfs [V12]), `system/limits`, `logs/rotation`, `security/lsm`.

### (e) Keep out (or not until the model changes)

| Utility | Reason |
|---|---|
| `vi`, `nano`, `less`, `htop`, `top` interactive, `crontab -e`, `visudo`, `passwd` | need a TTY / interactive; `files/update` and `cron/manage` are the replacements |
| `su`, `sudo -i`, `ssh`, `nsenter`, `chroot`, `gdb` | arbitrary shells or privilege pivots; defeats scoping |
| Generic "run this command" (`bash -c`) | the whole scoping model is per-tool and per-path; `docker/exec` is bounded by container grants, a host equivalent has no bound |
| `strace`, `perf record`, `tcpdump`, `nmap` | ptrace / packet capture / scanning third parties; unbounded output; too powerful |
| `journalctl -f`, `tail -f`, `watch`, `ping` (raw, continuous) | streaming/unbounded; one-shot with limits already exists |
| `mkfs`, `dd`, `parted`, `fdisk` (write), `wipefs`, `cryptsetup` | destructive with no undo |
| firewall / `ip addr|route` writes, `ip link set down` | can lock the agent (and the owner) out of the host; RCE-adjacent |
| `useradd/userdel/usermod`, `chpasswd` | credentials and account takeover; only as a later, heavily scoped ticket |
| `sudoers` writes | direct privilege escalation, self-lockout |
| `rsync`, `scp`, `wget` to remote hosts (beyond `network/curl`) | data exfiltration channel; needs CLI |
| `reboot`, `shutdown` | needs an approval/confirmation model first |
| `kubectl` | separate API, separate security domain |
| `apt install/remove`, `dnf` (v1) | long-running, runs maintainer scripts as root = arbitrary code |
| `at` with arbitrary command | same risk class as cron with a weaker audit trail; use transient timers |
| kernel `modprobe/insmod` | loading code into the kernel; only with a module-name allowlist, deferred |

### (f) Evidence

**T1 - baseline (target: repo code; the live `tools/list` on a stand was not queried).** Method: read the registry, cross-checked against the worker dispatch map.
- `grep -c '"name":          "[a-z]*/[a-z_-]*"' internal/rpc/tools.go` -> `38`; `grep '"name":' internal/rpc/docker.go` -> 8 `docker/*` names (`containers, manage, logs, exec, images, volumes, networks, prune`). Total 46.
- `cmd/mcpd/main.go` `handlers` map has additionally: `files/stat`, `files/content`, `files/read`, `processes/read`, `services/status`, `docker/inspect`, `read_usb|pci|dmi|modules|routes|interfaces` (internal workers, not in `tools/list`).
- Resources: 11 static + 10 templates listed in `internal/rpc/resources.go`.
- Ticket assumptions vs code: (1) "systemd timers ... already reachable over DBus like `services/*`" - **not yet**: `internal/tools/services/list/list.go` skips every unit not ending in `.service`; DBus itself can (V7). (2) "`auth/sudo-rules`" reads mcp-sudo.yaml only (README: "Reads the authorized privileged tools for the user from mcp-sudo.yaml") - the comment in `configs/mcp-sudo.yaml` (`auth/sudo-rules: # read the host's sudoers rules`) is inaccurate. (3) ARCHITECTURE.md lists `disks/fdisk`, `disks/smartctl`, `network/traceroute` - none exist in the registry; the real names are `disks/partitions` (native), `disks/health` (smartctl), `network/trace-path` (traceroute). (4) `internal/tools/files/get_open_files/` is an empty directory, not wired. (5) `files/find` also wraps a binary (`find`), not listed in the ARCHITECTURE exceptions I looked at (grep of ARCHITECTURE.md "exception|smartctl|traceroute" found smartctl, fdisk, traceroute only). No tool named in the ticket's "done" list is missing: files, disks, processes, network (`ss` = `network/connections`), memory, cpu, services, system, logs, kernel, users and docker all exist.

**T2 - native-source claims (target: VPS 89.125.210.117, Ubuntu 24.04 LTS, read-only commands via `ssh -i ~/.ssh/hostkey-usa-rsa root@...`; nothing written; the `amnezia-awg2` container and `/etc/shadow`, token and mcpd user files were not touched).** Excerpts trimmed.
- **V1** `grep PRETTY /etc/os-release; uname -r` -> `PRETTY_NAME="Ubuntu 24.04 LTS"`, `6.8.0-142-generic`.
- **V2** `cat /etc/crontab; for f in /etc/cron.d/*; do cat $f; done` -> `SHELL=/bin/sh`; `17 *	* * *	root	cd / && run-parts --report /etc/cron.hourly`; ... `@reboot root /opt/scan.sh`; drop-ins `atop`, `e2scrub_all`, `sysstat` (`5-55/10 * * * * root command -v debian-sa1 > /dev/null && debian-sa1 1 1`, `PATH=` env line). `ls -l /opt/scan.sh` -> `-rwxr-xr-x 1 root root 230 Jun 27 2024` (contents not read).
- **V3** `ls -la /etc/cron.allow /etc/cron.deny` -> `No such file or directory` for both.
- **V4** `ls -la /etc/cron.d /etc/cron.daily ...` -> `/etc/cron.d`: `atop e2scrub_all .placeholder sysstat`; `/etc/cron.daily`: `apport apt-compat dpkg logrotate man-db sysstat` (all `-rwxr-xr-x`) plus `.placeholder`; `cron.hourly`/`monthly` only `.placeholder`.
- **V5** `systemctl is-active cron atd anacron` -> `active`, `inactive`, `inactive`.
- **V6** `stat -c "%A %U:%G %n" /var/spool/cron/crontabs /usr/bin/crontab` -> `drwx-wx--T root:crontab /var/spool/cron/crontabs`, `-rwxr-sr-x root:crontab /usr/bin/crontab`; as `nobody` (`runuser -u nobody -- ...`): `cat /etc/crontab | wc -l` -> `24`, `cat /etc/cron.d/atop | wc -l` -> `4`, `ls /var/spool/cron/crontabs` -> `Permission denied`. The spool dir is empty for root (`ls -la` shows only `.` and `..`): no user has a crontab on this host.
- **V7** `systemctl list-timers --all` -> 18 timers (e.g. `apt-daily.timer  apt-daily.service`, next `10:12:15`, last `20:40:39`; `apport-autoreport.timer` with `-`). `busctl --system call org.freedesktop.systemd1 /org/freedesktop/systemd1 org.freedesktop.systemd1.Manager ListUnitsByPatterns asas 0 1 "*.timer"` -> `a(ssssssouso) 18 "fstrim.timer" "Discard unused filesystem blocks once a week" "loaded" "active" "waiting" ...`. `busctl get-property ... /unit/apt_2ddaily_2etimer org.freedesktop.systemd1.Timer TimersCalendar NextElapseUSecRealtime LastTriggerUSec Unit Persistent` -> `a(sst) 1 "OnCalendar" "*-*-* 06,18:00:00" 1790661600000000`, `t 1790676735137692`, `t 1790628039650769`, `s "apt-daily.service"`, `b true`. `busctl introspect ... Timer` lists properties `TimersCalendar, TimersMonotonic, NextElapseUSecRealtime, NextElapseUSecMonotonic, LastTriggerUSec, Persistent, RandomizedDelayUSec, Result, Unit`. `Manager` introspection shows `StartTransientUnit ssa(sv)a(sa(sv)) o`. Same property read as `nobody` succeeded (`t 1790676735137692`, `s "apt-daily.service"`).
- **V8** `ls /var/spool/cron/atjobs /var/spool/at /etc/at.allow /etc/anacrontab /var/spool/anacron; which at atq anacron cron crontab` -> all `No such file or directory`; `which` printed only `/usr/sbin/cron`, `/usr/bin/crontab`.
- **V9** `ls -la /etc/sudoers /etc/sudoers.d` -> `-r--r----- 1 root root 1800 /etc/sudoers`; `nobody` `cat /etc/sudoers` -> `Permission denied`; `grep -vE '^\s*(#|$)' /etc/sudoers` (root) -> `Defaults env_reset ... root ALL=(ALL:ALL) ALL ... %sudo ALL=(ALL:ALL) ALL`, `@includedir /etc/sudoers.d`. `sshd_config` -> `Port 22`, `PermitRootLogin yes`; `/etc/ssh/sshd_config.d/50-cloud-init.conf` exists (0600 root); `/root/.ssh/authorized_keys` has 1 line (`wc -l`; key material not printed).
- **V10** `busctl --system get-property org.freedesktop.timedate1 ... Timezone NTP NTPSynchronized CanNTP` -> `s "Etc/UTC"`, `b true`, `b false`, `b true` (NTP enabled, not synchronized at that moment); `org.freedesktop.hostname1 ... Hostname StaticHostname Chassis` -> `"213625.com" "213625.com" "vm"`; `busctl call org.freedesktop.login1 ... ListSessions` -> `a(susso) 1 "1156" 0 "root" ...`; `timedatectl show-timesync` -> `FallbackNTPServers=ntp.ubuntu.com ... RootDistanceMaxUSec=5s`; `/etc/systemd/timesyncd.conf` exists.
- **V11** `/proc/mdstat` -> `Personalities : [raid0] [raid1] [raid6] [raid5] [raid4] [raid10]` / `unused devices: <none>`; `/proc/swaps` -> `/swapfile file 524284 36288 -2`; `/proc/pressure/cpu` -> `some avg10=2.00 avg60=1.25 ...`, `/proc/pressure/memory` present; `/proc/modules` first line `tcp_diag 12288 0 - Live 0x...`; `/etc/modprobe.d` has `blacklist*.conf`; `/sys/kernel/security/lsm` -> `lockdown,capability,landlock,yama,apparmor`; `aa-status` present, `auditctl`/`ausearch` absent; `/var/run/reboot-required` -> `No such file` (none pending); `/var/lib/apt/periodic` has `update-success-stamp`, `unattended-upgrades-stamp`, ...; `/etc/security/limits.conf` has no active lines; `/sys/fs/cgroup/cgroup.controllers` present (cgroup v2); `/etc/apt/preferences.d` lists `ubuntu-pro-esm-*`; `ls /sys/block/dm-*` -> none.
- **V12** `/sys/class/net/<if>/{operstate,mtu,speed}` -> `up 1500 10000` (a virtual interface); `/proc/net/route` header + default via `eth0` present; `/proc/net/nf_conntrack` -> absent; `/proc/net/netfilter` has only `nf_log`; `/proc/net/ip_tables_names` printed nothing (not readable/absent - unverified as a source); `which nft iptables ufw` -> all three present; `which podman virsh kubectl lxc` -> only `/usr/sbin/lxc`; `/sys/class/hwmon` empty, `/sys/class/thermal/cooling_device0` only; `/var/run/docker.sock` present.
- **V13** `ls -la /etc/passwd /etc/group /etc/login.defs /etc/environment` all 0644; `getent passwd | wc -l` -> `40`; `getent group crontab` -> `crontab:x:990:`; `/etc/fstab` 2 active lines (ext4 `/` + `/swapfile`); `systemctl list-units --failed` -> `ifup@eth0.service`, `networking.service` failed; `busctl ... Manager ListUnitFiles` -> `a(ss) 410 ...` (410 unit files); `Manager` introspection has `Reload`, `EnableUnitFiles`, `LinkUnitFiles`.
- **V14** `ls -la /var/log/wtmp /var/log/btmp /var/log/auth.log` -> `wtmp -rw-rw-r-- root utmp 19584`, `btmp -rw-rw---- root utmp 141598080` (root/utmp only), `auth.log -rw-r----- syslog adm`; `du -sh /var/log/journal` -> `1.5G`; `/etc/logrotate.d` contains `alternatives apport apt bootlog btmp cloud-init dpkg ... ufw unattended-upgrades`.
- **V15** `ls /etc/ssl/certs | wc -l` -> `243`; `/etc/letsencrypt/live` -> `No such file`; `which openssl` -> `/usr/bin/openssl`.
- **Unverified (no host or not testable read-only):** RHEL/cronie spool path `/var/spool/cron/<user>` and RHEL allow/deny defaults; exact cron.d filename/ownership rules and dot-suffix disabling; Debian cron mtime re-read behaviour and "drop whole crontab on syntax error"; `at` job file format; `/var/spool/anacron` layout; `crontab`'s own validation flags; SELinux paths (`/sys/fs/selinux/enforce`); quota, PAM faillock and fail2ban sources; nftables via netlink from Go without a library (assessed as impractical, not attempted); non-root read of `/proc/net/ip_tables_names`; local k8s node (not checked, the VPS was the only target; this is a gap for T2 as written in the ticket).


## Comments

- 2026-09-29 - created in `new/` at the owner's request; research not started. IDs FR-021 (linuxctl edit: validate even when the editor changed nothing, plus a `validate` verb) and this one are the newest; FR-021 has not been filed yet.
- 2026-09-29 - research done (no code, no other files): baseline from the code registry (46 tools, 11 static resources, 10 templates), 83-row inventory, cron/timers/at deep-dive with proposed schemas for `cron/list`, `cron/manage`, `timers/list`, ranked list of 15 tools, keep-out list, evidence V1-V15 gathered read-only on the VPS (and one non-root `runuser -u nobody` check of file/DBus readability). `amnezia-awg2`, shadow, tokens and mcpd users.yaml were not touched. Unverified: RHEL spool paths/allow-deny defaults, exact cron.d name/ownership rules, Debian cron reload and syntax-error behaviour, `at` job format, anacron layout, SELinux/quota/PAM/fail2ban sources, nftables without a library, the local k8s node. Noted for the owner: `@reboot root /opt/scan.sh` in the VPS `/etc/crontab` (non-stock, contents not read); the ticket's premise that timers are already reachable is not true yet (`services/list` filters `.service`); the `privileged` block comment on `auth/sudo-rules` and ARCHITECTURE.md tool names are inaccurate. Acceptance criteria left unticked for the owner; T3/T4 are the owner's.
- 2026-09-29 - reviewed the agent's write-up before committing: it changed only this file (+341 lines). Two of its findings re-checked by hand: (1) ARCHITECTURE.md lines 52 and 55 name `disks/fdisk`, `disks/smartctl` and `network/traceroute`, which do not exist (`internal/tools/disks` has free health list mounts partitions performance usage; `internal/tools/network` has arp connections curl nslookup ping trace-path) - the doc is stale, worth its own small fix; (2) the VPS `/etc/crontab` really has `@reboot root /opt/scan.sh` (script 230 bytes, root:root 0755, dated 2024-06-27; crontab last modified 2026-06-12) - not read; probably part of the provider's image, but the owner should look. Left for the owner: accept/reject the ranked list in section (d) and the safety decisions in (c); nothing is ticked, no follow-up tickets filed yet.

