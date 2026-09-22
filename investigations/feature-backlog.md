# Feature Backlog — Synthesized from `investigations/*.md`

Every item here is grounded in a specific scenario actually tested live against the deployed daemon (cross-referenced by file/section below) — not a wishlist. Ordered by how many distinct tested scenarios each would unblock, then by severity. See individual category files for full evidence and real output.

## Bugs in existing tools (fix, don't build)

### 1. `processes/list`'s `sort_by: cpu` is non-functional
**Evidence**: `cpu-performance.md` #2. The schema accepts `sort_by: cpu`, but there's no CPU field on the returned object at all, and the sort silently falls through to unsorted order — confirmed by comparing against working `sort_by: pid` (correctly ascending) and `sort_by: mem` (correctly descending).
**Impact**: Blocks `cpu-performance.md` #2 (the single most common "server is slow" question) and weakens #6, `containers-kubernetes.md` #3.
**Fix**: Add per-process CPU% (sampled via two `/proc/[pid]/stat` `utime`+`stime` reads a short interval apart, same technique `top` uses) to the `Process` struct, and implement the `cpu` case in the sort switch. This is the single highest-value item in this backlog.
**Test plan**: Call `processes/list` with `sort_by: cpu` twice, once at idle and once while a CPU-bound process runs (e.g. spin up a `yes > /dev/null` on the real host via a privileged call, or use an existing busy process); confirm the busy process sorts to the top with a plausible non-zero CPU% distinct from `mem`-sorted order.

### 2. `network/connections`'s `state` filter rejects standard `ss` state names
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

- **11. `files/list` ownership/mode fields** (`users-auth.md` #1) — add UID/GID/mode to the existing output rather than requiring a separate stat call.
- **12. `dmesg` `since`/`until` filters** (`logs-diagnostics.md` #3) — bring it to parity with `journal-control`'s existing time filters.
- **13. `journalctl -b` (since-last-boot) shortcut** (`boot-kernel-hardware.md` #5) — a `boot: true`/`boot_offset` parameter on `logs/journal-control`.
- **14. cgroup-aware CPU/memory stats** (`cpu-performance.md` #5, `containers-kubernetes.md` #3) — read `/sys/fs/cgroup/*/cpu.stat` and `memory.current` for per-container/per-slice attribution; natively parseable, no new dependency.
- **15. "Which package provides this binary" lookup** (`packages-updates.md` #4) — cross-reference `dpkg -S`-equivalent (parse `/var/lib/dpkg/info/*.list`) against a binary path.
- **16. Package install/remove** (`packages-updates.md` #3, roadmap item 4) — needs the dry-run/confirmation governance layer roadmap item 5 already flags as a prerequisite; don't ship standalone.
- **17. `auth/ssh-keys`** (`users-auth.md` #3, roadmap item 1).
- **18. Account lock/expiry status without touching password hashes** (`users-auth.md` #4) — needs its own careful scoping (expose only non-hash `/etc/shadow` fields), not a blanket shadow-read.
- **19. Swap thrashing rate** (`memory.md` #5) and **disk I/O live rate** (`disk-storage.md` #6) — both need two time-separated samples and a delta calculation; currently only cumulative-since-boot counters are exposed.

## Explicitly not recommended (from `plan/linux-admin-roadmap.md`, reaffirmed by this investigation)

Disk partitioning (write) and firewall rules (write) both remain correctly deferred pending the dry-run/confirmation governance design — nothing in this investigation's live testing changed that assessment; if anything, the real, currently-full disk found in `disk-storage.md` makes it *more* important that any future write-capable disk tool ships with strong safeguards from day one, not less.
