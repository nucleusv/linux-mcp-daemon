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
| T1 | Baseline is accurate | list every tool and resource from `get mcp-api tools/resources` on a stand and diff against the "covered today" column | no covered tool missing from the table, no listed tool that does not exist | | |
| T2 | Native claims are real | for each "native source" claim, check on a real host (VPS 89.125.210.117, local k8s node) that the file/DBus/netlink source exists and yields the data | each claim backed by real output or marked "needs the CLI" | | |
| T3 | Proposal reviewed | the owner reads the ranked list | accepted / rejected / deferred per item, recorded here | | |
| T4 | Follow-ups filed | one ticket per accepted item in `backlog/new/` | each has description, criteria and tests | | |

## Comments

- 2026-09-29 - created in `new/` at the owner's request; research not started. IDs FR-021 (linuxctl edit: validate even when the editor changed nothing, plus a `validate` verb) and this one are the newest; FR-021 has not been filed yet.
