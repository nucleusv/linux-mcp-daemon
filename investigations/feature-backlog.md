# Feature Backlog — Synthesized from `investigations/*.md`

Every item here is grounded in a specific scenario actually tested live against the deployed daemon (cross-referenced by file/section below) — not a wishlist. Ordered by how many distinct tested scenarios each would unblock, then by severity. See individual category files for full evidence and real output.

## Bugs in existing tools (fix, don't build)

### 1. ~~`processes/list`'s `sort_by: cpu` is non-functional~~ — done (5f9d96d: two-sample CPU% via `internal/procstat`, shared with `processes/top`)
**Evidence**: `cpu-performance.md` #2. The schema accepts `sort_by: cpu`, but there's no CPU field on the returned object at all, and the sort silently falls through to unsorted order — confirmed by comparing against working `sort_by: pid` (correctly ascending) and `sort_by: mem` (correctly descending).
**Impact**: Blocks `cpu-performance.md` #2 (the single most common "server is slow" question) and weakens #6, `containers-kubernetes.md` #3.
**Fix**: Add per-process CPU% (sampled via two `/proc/[pid]/stat` `utime`+`stime` reads a short interval apart, same technique `top` uses) to the `Process` struct, and implement the `cpu` case in the sort switch. This is the single highest-value item in this backlog.
**Test plan**: Call `processes/list` with `sort_by: cpu` twice, once at idle and once while a CPU-bound process runs (e.g. spin up a `yes > /dev/null` on the real host via a privileged call, or use an existing busy process); confirm the busy process sorts to the top with a plausible non-zero CPU% distinct from `mem`-sorted order.

### 2. ~~`network/connections`'s `state` filter rejects standard `ss` state names~~ — done (3a7c7ac: case-insensitive mapping to ss names; all sockets listed by default)
**Evidence**: `network.md` setup (before scenario 1). `state: "LISTEN"` fails with `ss: wrong state name: LISTEN`; omitting the filter and reading full output works fine and even the tool's own listed output literally shows the string `LISTEN` in the `State` column.
**Fix**: Either lowercase the value before passing to `ss` (`ss` expects `listening`, not `LISTEN`), or document the exact accepted values in the schema description so callers don't guess wrong from the tool's own output format.
**Test plan**: Call with `state: "listening"` (lowercase) and confirm it returns only `LISTEN` rows; call with the currently-broken `"LISTEN"` and confirm it now either works or gives a clear "did you mean" error instead of a raw `ss` stderr dump.

### 3. `network/nslookup` hangs for the full worker timeout on unresolvable names instead of failing fast
**Evidence**: `network.md` #4. Looking up a real, valid but unreachable-from-here name (`kubernetes.default.svc.cluster.local`, a cluster-internal DNS zone this `hostNetwork: true` daemon structurally cannot reach) hung for the entire 30-second worker timeout rather than returning NXDOMAIN/an error quickly.
**Fix**: Set a short internal DNS query timeout (a few seconds) independent of the worker's overall timeout, so a resolution failure surfaces as a fast, clear error instead of looking like a hung tool.
**Test plan**: Repeat the same real cluster-internal lookup; confirm it now returns within ~5s with a clear "could not resolve" error rather than the 30s worker-timeout message.

### 4. Missing binaries reachable only via `privileged: true`, with no upfront signal
**Evidence**: `disk-storage.md` #4 (`fdisk`), `services-systemd.md` #3 / `logs-diagnostics.md` #2 (`journalctl`, works with `privileged: true`, fails without — worse, since it's *usable* with the flag, just silently mandatory), earlier this session (`lastb` on the real host).
**Fix, split in two**: (a) add `fdisk` to the Docker image (a genuine bug — the tool is documented as always-wrapping-fdisk with no alternative), matching how `file` was added earlier this session when `files/filetype` hit the same class of issue. (b) For `journalctl` specifically: since it's *architecturally* host-only (never in this daemon's own image by design), update the tool's schema description to say so explicitly (`"privileged: true" is required, not optional, in containerized deployments`), so the failure mode without it is expected rather than confusing.
**Test plan**: Rebuild the image; call `disks/partitions` on a real device without any special flag and confirm it now returns real partition data instead of the exec error. For journalctl, confirm the updated tool description surfaces the requirement in `tools/list` output.

## New tools — read-only, high value

### 5. System-wide deleted-but-open file scan
**Evidence**: `disk-storage.md` #3 — a **live, real, currently-unsolved** ~50 GiB "disk full but `du` doesn't add up" mystery on this exact host during this investigation. `process://{pid}/open_files` can confirm/deny one PID at a time but there's no bulk scan.
**Design**: A new tool (e.g. `processes/open-deleted-files` or `disks/deleted-files`) that iterates every PID's `/proc/[pid]/fd/*` natively (same `os.ReadDir`+`os.Readlink` technique `open_files` already uses, just looped over every running PID instead of one) and returns every `(deleted)` target with an estimated size (`stat` the fd via `/proc/[pid]/fd/N`, which still reports the file's size even after unlink), sorted descending by size.
**Test plan**: Run it against this same host, right now, while the disk is still at 100%; it should either find the real ~50 GiB culprit (closing the loop this investigation left open) or definitively rule out "deleted-but-open" as the cause, which is itself a valuable, currently-impossible-to-get answer.

### 6. Firewall rules (read-only first)
**Evidence**: `network.md` #5, `security-access.md` #3. Confirmed via `system/packages` that the real host has both `iptables` and `nftables` installed — not a dead end, there's real data to read.
**Design**: Wrap `iptables-save`/`nft list ruleset` (both plain-text, following the project's documented CLI-wrap precedent), matching `plan/linux-admin-roadmap.md` item 3's existing proposal.
**Test plan**: Call it against the real host; confirm the output includes at least the `KUBE-*` chains kube-proxy is known to install on this exact cluster (visible indirectly via the `kube-proxy` process already confirmed running).

### 7. systemd timer listing
**Evidence**: `services-systemd.md` #4 — resolves `plan/linux-admin-roadmap.md` item 2's open question with real evidence: `services/list` returns zero non-`.service` units even unfiltered, confirmed not a filtering artifact.
**Design**: Extend `services/list` (or add `services/timers`) to also query `.timer` unit type via the same `go-systemd/v22/dbus` connection already in use.
**Test plan**: Call it against a host known to have timers (this specific kind node may genuinely have none — verify against a standard Ubuntu box, e.g. via a local test container, where `apt-daily.timer`/`systemd-tmpfiles-clean.timer` are expected to exist by default).

### 8. SUID/SGID scan
**Evidence**: `security-access.md` #1 — confirmed via direct schema inspection that `files/find` has no mode/permission filter at all.
**Design**: Native `filepath.WalkDir` + checking `info.Mode()&os.ModeSetuid != 0` (same technique `disks/usage` already uses for tree-walking, zero new dependencies), per `plan/linux-admin-roadmap.md` item 7's proposal.
**Test plan**: Run against `/usr/bin` on the real host; confirm it finds known-real SUID binaries (e.g. `passwd`, `sudo` if present) and excludes non-SUID ones.

### 9. Pending security updates
**Evidence**: `packages-updates.md` #2, matches roadmap item 7.
**Test plan**: Compare its output against `apt list --upgradable` run manually via `files/read`-of-nothing (i.e. cross-check with a raw shell on the same host) to confirm parity.

### 10. Container/pod-aware listing (`containers/list`)
**Evidence**: `containers-kubernetes.md` #1, #2 — a real, live host running real containerd-managed containers that are currently invisible as anything other than opaque `containerd-shim` host processes.
**Design**: Native containerd gRPC client (kernel-first-equivalent per `ARCHITECTURE.md`'s own framing), not shelling to `crictl`.
**Test plan**: Run against this exact host; confirm it correctly enumerates the same containers whose shims are visible in `processes/list` today (cross-reference shim `-id` values in `cmdline` against the new tool's container IDs).

## Smaller/lower-priority items (single-scenario, still real)

- ~~**11. `files/list` ownership/mode fields**~~ — done (2bde2c0: `files/list` is now `ls -la`, with mode/owner/group in text and JSON) (`users-auth.md` #1) — add UID/GID/mode to the existing output rather than requiring a separate stat call.
- **12. `dmesg` `since`/`until` filters** (`logs-diagnostics.md` #3) — bring it to parity with `journal-control`'s existing time filters.
- ~~**13. `journalctl -b` (since-last-boot) shortcut**~~ — done (`logs/journal-control` has `boot` / `boot_offset`) (`boot-kernel-hardware.md` #5) — a `boot: true`/`boot_offset` parameter on `logs/journal-control`.
- **14. cgroup-aware CPU/memory stats** (`cpu-performance.md` #5, `containers-kubernetes.md` #3) — read `/sys/fs/cgroup/*/cpu.stat` and `memory.current` for per-container/per-slice attribution; natively parseable, no new dependency.
- **15. "Which package provides this binary" lookup** (`packages-updates.md` #4) — cross-reference `dpkg -S`-equivalent (parse `/var/lib/dpkg/info/*.list`) against a binary path.
- **16. Package install/remove** (`packages-updates.md` #3, roadmap item 4) — needs the dry-run/confirmation governance layer roadmap item 5 already flags as a prerequisite; don't ship standalone.
- **17. `auth/ssh-keys`** (`users-auth.md` #3, roadmap item 1).
- **18. Account lock/expiry status without touching password hashes** (`users-auth.md` #4) — needs its own careful scoping (expose only non-hash `/etc/shadow` fields), not a blanket shadow-read.
- **19. Swap thrashing rate** (`memory.md` #5) and **disk I/O live rate** (`disk-storage.md` #6) — both need two time-separated samples and a delta calculation; currently only cumulative-since-boot counters are exposed.

## Daemon infrastructure

### 20. ~~Leveled, configurable daemon logging~~ — done (log/slog, `logging:` in daemon.yaml, audit + access lines at any level, runtime change via daemon/reload-config)
**Now**: 27 plain `log.Printf` calls across 5 files - no levels, no configuration, ad-hoc prefixes (`[ACCESS]`, `[SECURITY]`, `[THROTTLED]`, `[TOOL CALL]`, `WARNING:`). Every request writes several lines (connect, `Received JSON-RPC`, access line, response size, disconnect), which is noise on a busy host - the VPS journal was dominated by it - yet there's no way to turn detail *up* when debugging either.
**Build**: stdlib `log/slog` (keeps the zero-dependency rule), configured in `daemon.yaml`:
```yaml
logging:
  level: info        # error | warn | info | debug
  format: text       # text | json (json for journald/Loki/ELK field extraction)
  access_log: true   # one line per HTTP request, independent of level
```
Suggested mapping of what exists today:
- **error**: worker spawn failures, config errors, TLS/listen failures.
- **warn**: `[SECURITY]` (session hijack attempts), `[THROTTLED]`, containerized-mode mismatches, denied privileged calls, worker timeouts.
- **info**: startup/shutdown with version and config summary, one line per tool call (user, tool, privileged, duration, ok/error), and an **audit** line for every mutating call - `files/create|update|chmod|chown`, `processes/delete`, `services/manage`, sysctl writes - that must stay on even when detail is turned down (a separate `audit` attribute or logger, not tied to the level).
- **debug**: per-request JSON-RPC method/id, SSE session lifecycle, cache hits/misses, rate-limiter decisions, worker exit codes and stderr.
Carry structured fields (`user`, `session`, `tool`, `privileged`, `duration_ms`, `exit`) rather than formatted strings. Keep the existing safeguards: arguments go through `redactArgs`, responses are logged by size only - **debug must never dump tool output or tokens**. Optional: change the level at runtime (SIGHUP re-reading `logging`, or a `linuxctl` admin command) so debugging a live host doesn't need a restart.
**Test plan**: at `info`, a tool call writes exactly one call line (plus one audit line if mutating) and no per-request JSON-RPC lines; at `debug`, the session lifecycle and RPC lines appear; at `warn`, a throttled or hijack attempt still logs while normal calls don't; `format: json` output parses line by line; grep every level's output for a known test token and a known file's contents to prove neither leaks.

### 21. Native `files/find` (no find(1)), parallel, with the same filters
**Now**: `files/find` runs the system `find` with validated arguments (`-P`, never follows symlinks; with `_no_follow` the start directory is opened without symlinks and searched as `.`). It works, but it's one of the last tools wrapping a binary.
**Build**: a Go walker with the same filters (`name` glob, `type`, `mtime`, `size`, `max_depth`, virtual-FS pruning), walking directories concurrently for speed comparable to GNU find, and descriptor-relative (openat with O_NOFOLLOW, as `internal/fsafe`) so a directory swapped for a symlink mid-walk can't redirect it.
**Status**: deferred at the user's request (2026-09-24) - a first attempt was interrupted by the assistant's safety filter; the user sent feedback about it. Try again later.
**Test plan**: same results as `find` (sorted paths) on a host tree and on generated trees with every filter; symlink loops and swapped directories never escape the start directory; runtime within ~2x of GNU find on a large tree.

### 22. ~~README: what the daemon is for~~ — done (README "Why" section)
**Now**: README says what mcpd does, not why someone would run it.
**Build**: a short "Why / use cases" section near the top: incident triage and on-call (an agent reading disks, processes, sockets, journal - structured, as the calling user); audits and inventory across hosts; giving an agent *bounded* root (per tool, per path, per sysctl key, per network destination) instead of SSH root or a shell MCP server; every change audit-logged. Plus what it deliberately isn't (no arbitrary shell, no file deletion) and how it compares to SSH-shell MCP servers and osquery.

### 23. ~~Docs: a page on mcp-sudo defaults and the risks of granting root~~ — done (`configuration/permissions-and-risks.md`, linked from mcp-sudo, installation and README; also closed the uid 0 MCP-user hole it turned up)
**Now**: `configuration/mcp-sudo.md` describes the options; the defaults and their consequences are spread over it.
**Build**: a dedicated page - what a user can do with no entry, with `allowed` only, with `paths`, `network`, `sysctl` (the defaults table), each shown on a concrete config and the calls it allows/refuses. Then the risks, bluntly: which grants amount to full root (`files/update`/`create` on `/`, `files/chmod`/`chown` on `/`, `services/manage`, `kernel/system-control` without `write_keys` - e.g. `kernel.core_pattern`, `processes/delete`, `file://` `""` reads `/etc/shadow`), least-privilege recipes for common roles (read-only diagnostics, log reader, web-server operator), and a checklist before granting. The same warning, short, in README and installation.

### 24. ~~Docs: tool and linuxctl output captured on a systemd host~~ — done (recaptured on an Ubuntu 24.04 VPS, 2026-09-26)
**Now**: most MCP API and linuxctl pages show output captured in the Kubernetes dev node (a minimal LinuxKit VM: no DMI, no `last`, container paths).
**Build**: recapture every tool page's linuxctl and raw JSON-RPC output on a real systemd host (the VPS), without secrets (no /etc/shadow, no client IPs, no tokens), and note the host once per page.

### 25. ~~TLS on by default~~ — done (`server.tls` on 9091 with a generated self-signed cert, plain HTTP off; linuxctl fingerprint/CA trust)
**Now**: plain HTTP unless `server.tls` is configured by hand with an existing certificate; bearer tokens travel in clear text.
**Build**: TLS enabled by default: install.sh / the packages / the container generate a self-signed certificate (or take a provided one / ACME) on first install; `linuxctl` trusts it via a pinned CA or fingerprint; plain HTTP only when explicitly enabled (e.g. behind a TLS-terminating proxy). In progress (2026-09-24).

## To investigate: similar projects

### 26. Compare with other Linux MCP servers
**Links** (from the user, 2026-09-25):
- [marcos2872/os-mcp](https://lobehub.com/zh/mcp/marcos2872-os-mcp#1) (LobeHub)
- [Mohabdo21/linux-mcp](https://glama.ai/mcp/servers/Mohabdo21/linux-mcp#get_docker_system_snapshot) (Glama) - note its `get_docker_system_snapshot` tool

**Do**: list each one's tools and resources, how it reaches the host (shell commands vs `/proc`/`/sys`), how it limits what an agent may do (auth, root, per-tool grants), and transport. Note tools they have that we lack (containers/Docker is our open item 10) and anything worth borrowing; also how they are listed and described on these directories - input for publishing mcpd there and for the Habr article.

**Findings (2026-09-26, read-only review of both repos):**
- **os-mcp** (Rust, stdio): two tools - `get_system_info` and `execute_command`, i.e. a remote shell. `sh -c` with an allowlist checked on the first word only (`ls; rm -rf ~` passes), 227 allowed commands incl. `env`, `awk`, `curl`, `systemctl`, root via `pkexec sh -c` (needs a desktop). No auth, no releases, 4 stars.
- **Mohabdo21/linux-mcp** (Go, stdio): 69 read-only `get_*` tools + 19 resources via gopsutil, `/proc`, the Docker SDK and `exec` without a shell. No auth, no root model (runs as the client). Tool disable list + SIGHUP reload. Publishes to the MCP Registry (`server.json`) and Glama (`glama.json`). One tool calls ip-api.com.
- **Neither** has authentication, network transport, per-user identity, per-call privilege drop, per-tool root grants, path/network limits, or CI running tests - what sets mcpd apart.

**Worth adding, ranked:**
1. Read-only Docker group (open item 10): containers, logs (tail), stats, inspect with secret redaction, disk usage, and one `docker/snapshot` that returns partial results with an `errors[]` field instead of failing.
2. A one-call system health summary (load, memory, disks, failed units, recent errors) - saves an agent many round-trips.
3. A security audit tool: SUID and world-writable files, firewall, sshd config, SELinux/AppArmor, failed logins - root only where granted.
4. Boot analysis (blame, critical chain) from systemd over D-Bus.
5. Smaller: cron jobs and timers, available package updates, process tree and open files, largest files, CPU temperature.

**Listing mcpd:** add `glama.json` and an MCP Registry `server.json`; tagline naming the difference (remote over HTTPS, per-user privilege-isolated workers, per-tool root grants); categories Monitoring, System Administration, Security; tool/resource counts and license.

### 27. `disks/usage`: path first, then size
**Now**: text output is du's layout, size then path (`79519744	/var/log`) - see https://nucleusv.github.io/linux-mcp-daemon/next/mcp-api/tools/disks/usage.
**Do** (user, 2026-09-26): print the path first and the size after it, in the code; then recapture the docs page (and the linuxctl command reference's `disks usage` example).

## Explicitly not recommended (from `plan/linux-admin-roadmap.md`, reaffirmed by this investigation)

Disk partitioning (write) and firewall rules (write) both remain correctly deferred pending the dry-run/confirmation governance design — nothing in this investigation's live testing changed that assessment; if anything, the real, currently-full disk found in `disk-storage.md` makes it *more* important that any future write-capable disk tool ships with strong safeguards from day one, not less.
