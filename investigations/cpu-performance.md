# CPU & Performance Troubleshooting

## 1. "Load average is high, walk me through what you check" (classic SRE interview question)

**Ticket**: Alert fires: load average elevated on the node.

**Real test**: `cpu/load-average` returned `Load Average: 0.15, 0.61, 0.96` (1m, 5m, 15m). `cpu/list` confirms 4 processors. Load of ~1.0 on 15m against 4 cores is not actually concerning (25% of capacity), but the tool correctly gives the numbers needed to make that call, and the trend (0.15 → 0.61 → 0.96, i.e. rising over the last 15 minutes) is visible from the three numbers alone.

**Verdict: Solvable** for the "what's the number and is it trending up" half of this question.

## 2. Find which process is actually consuming CPU (the natural next step, and its own extremely common interview question)

**Ticket**: Load is high — which process is responsible?

**Real test**: `processes/list` accepts a `sort_by: cpu` parameter (confirmed present in its schema). Called it live: it returned processes in the **same order as the natural/unsorted listing** (PID 1, 1031, 1156, 1265, 1297...) — not sorted by any CPU metric. Compared directly against `sort_by: pid`, which correctly produced strict ascending order (1, 151, 268...). The two are visibly different orderings, confirming `sort_by: cpu` is not silently equivalent to `sort_by: pid` — it does nothing (falls through to unsorted iteration order).

More fundamentally: **the `Process` JSON object itself has no CPU field at all** — only `pid`, `user`, `comm`, `state`, `ppid`, `rss_kb`, `cmdline`. There is no percentage, no cumulative CPU time, nothing to sort by even if the sort logic were fixed.

**Verdict: Not solvable.** This is the single most common "the server is slow" first question a Linux admin asks (`top`'s entire reason for existing), and it's a hard gap today — not a missing edge case, a missing core capability. Worth calling out as a real, misleading bug too: the tool schema advertises `sort_by: cpu` as a valid, accepted value, which will mislead any caller (human or AI) into believing this works.

## 3. Determine if the system is CPU-bound or I/O-bound

**Ticket**: High load average, but is it actually CPU contention, or processes stuck in uninterruptible I/O wait (`D` state)?

**Real test**: `processes/list` does return a `state` field (`S` = sleeping seen throughout this host's real process list). Filtering/counting processes in `D` state (uninterruptible sleep, the classic I/O-wait signature) is possible by reading this field from the returned JSON, though the tool has no built-in "count by state" or "show only D-state processes" filter — an admin/AI would do this client-side over the full list.

**Verdict: Partially solvable.** The raw signal (`state`) is present per-process, but there's no aggregate CPU-time-in-user-vs-iowait breakdown (the `/proc/stat` `cpu` line's `iowait` field isn't exposed anywhere) to answer "is the *system* I/O-bound" directly — only "is this *specific process* currently blocked."

## 4. Find zombie / defunct processes

**Ticket**: `ps` shows `<defunct>` processes accumulating; check if a parent is failing to reap children.

**Real test**: `processes/list`'s `state` field would show `Z` for a zombie process (standard `/proc/[pid]/stat` state codes), same mechanism as scenario 3. None were present on this host at test time, so this specific check couldn't be exercised against real zombie data, but the underlying data path (parsing `state` from `/proc/[pid]/stat`) is the same one already confirmed working for `S` state above.

**Verdict: Solvable** (by inspection of the existing `state` field, filtered client-side), though a dedicated `state: "Z"` filter parameter would make this a one-call answer instead of a manual scan.

## 5. Check CPU steal time / throttling in a virtualized or container-cgroup environment

**Ticket**: A workload's CPU allocation looks fine per `top`, but it's still slow — check for hypervisor steal time or cgroup CPU throttling.

**Real test**: No tool exposes `/proc/stat`'s per-CPU `steal` field, nor `/sys/fs/cgroup/*/cpu.stat`'s `nr_throttled`/`throttled_usec` fields. `cpu/list` only returns static topology (vendor, model, MHz, cache) from `/proc/cpuinfo` — none of it is live utilization data.

**Verdict: Not solvable.** No coverage at all today, and this is a genuinely common cause of "my container is slow but its host looks idle" confusion in exactly this daemon's own real deployment (Kubernetes-in-Docker, where cgroup CPU quotas are enforced per-pod).

## 6. "The website is down, walk me through your diagnosis" (classic SRE interview question, CPU/load angle)

**Ticket**: The generic version of this question always has a "check system resources" branch alongside the network/service branches (covered in their own category files).

**Real test**: Chaining `cpu/load-average` → `memory/usage` → `processes/list` (sorted by `mem`, since `cpu` doesn't work) gives a real, if CPU-blind, resource snapshot. On this host: load 0.15/0.61/0.96, memory 2.1 GB used of 8.3 GB total, 5.8 GB available — nothing pointing to resource exhaustion as the cause, which is itself a useful (negative) diagnostic result ruling out "the box is out of resources" as an explanation.

**Verdict: Partially solvable** — good for ruling resource exhaustion in/out at a glance, but incomplete without working per-process CPU attribution (scenario 2).
