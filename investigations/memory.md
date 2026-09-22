# Memory Troubleshooting

## 1. "Server ran out of memory, the OOM killer fired — what happened?" (classic SRE interview question)

**Ticket**: A process died unexpectedly; suspect the kernel OOM killer.

**Real test**: `logs/dmesg` filtered to `level: "err,warn"` returned real kernel messages from this host, but no actual OOM-killer event occurred during this investigation (no `"Out of memory"` / `"Killed process"` lines present) — genuinely nothing to find, not a tool failure. What the same call *did* surface, live: repeated `cfs_period_timer[cpuN]: period too short, scaling up` warnings — real evidence of cgroup CPU-quota pressure on this host, directly relevant to the CPU-throttling gap noted in `cpu-performance.md` scenario 5. `dmesg` happens to be a working side-channel for that even without a dedicated cgroup-stats tool.

**Verdict: Solvable, mechanism confirmed** — `logs/dmesg` can find and filter kernel-level OOM messages by severity level; just no real incident was live to test the exact OOM message format against. `logs/journal-control` (filtered by `since`/`unit`) is the other place `systemd-oom`/`earlyoom` events would show up, also already working.

## 2. Basic memory pressure check

**Ticket**: Is this host actually low on memory right now?

**Real test**: `memory/usage` (summary) on this host, live:
```
Total: 8.3 GB   Used: 2.1 GB   Free: 519 MB   Buff/Cache: 5.7 GB   Available: 6.0 GB
```

**Verdict: Solvable**, and correctly reports the metric that actually matters (`Available`, not `Free`) — see scenario 3.

## 3. "Free memory looks almost zero, is the system about to OOM?" (classic admin misconception, also a common interview trick question)

**Ticket**: `free -m` (or here, `memory/usage`) shows very little "free" memory; is that dangerous?

**Real test**: `memory/usage` with `detailed: true` on this host shows `MemFree: 502328 kB` (looks alarming in isolation) alongside `MemAvailable: 5820604 kB` and `Cached: 4240372 kB` — the classic Linux pattern where most "used" memory is reclaimable page cache, not a leak or pressure. The tool exposes both numbers side by side.

**Verdict: Solvable**, and correctly avoids the classic trap — an admin or AI reading this output has what's needed to answer "no, this is fine" correctly instead of panicking at a low `Free` number.

## 4. Which process is using the most memory

**Ticket**: Memory usage is high; which process is responsible?

**Real test**: `processes/list` with `sort_by: mem` correctly returned real, properly descending-sorted results: `kube-apiserver` (336 MB) → `kubelet` (126 MB) → `next-server` (120 MB) → `kube-controller-manager` (119 MB) → `containerd` (104 MB). Verified this is genuinely sorted (not coincidental) by cross-checking the RSS values are strictly decreasing.

**Verdict: Solvable**, and unlike the equivalent CPU question (`cpu-performance.md` scenario 2), this one actually works correctly today.

## 5. Check swap usage / thrashing

**Ticket**: System feels slow; is it swapping heavily?

**Real test**: `memory/usage` with `detailed: true` shows `SwapTotal: 1048572 kB`, `SwapFree: 1026020 kB` — about 22 MB of 1 GB swap in use on this host, i.e. negligible, not thrashing.

**Verdict: Solvable.** The raw numbers are there; there's no derived "swap-in/swap-out rate" (would need `/proc/vmstat`'s `pswpin`/`pswpout` deltas over time, not currently exposed), so detecting active *thrashing* (as opposed to static swap usage) would need repeated polling and manual delta calculation, same limitation as I/O rate in `disk-storage.md` scenario 6.

## 6. Detect a slow memory leak in a long-running process

**Ticket**: A service's memory usage climbs steadily over days; confirm and identify it.

**Real test**: `processes/list` gives a single point-in-time RSS snapshot per process (confirmed accurate in scenario 4). There's no built-in history/trend tracking.

**Verdict: Partially solvable.** An AI operator *can* solve this procedurally today — poll `processes/list` at intervals and diff RSS over time client-side — but the daemon itself has no time-series memory of past calls, so this requires the caller to do the tracking, not a single diagnostic call.
