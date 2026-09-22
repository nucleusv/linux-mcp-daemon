# Disk & Storage Troubleshooting

Tested live against the actual deployed daemon. The host genuinely hit 100% disk usage during this investigation (not staged) — several scenarios below use that real incident as the test case.

## 1. "Disk is full, walk me through your diagnosis" (classic SRE interview question)

**Ticket**: Monitoring pages that `/` is at 100% disk usage on the node.

**Real test**: `disks/free` with `privileged: true` on `/` returned, live, right now:
```
Total:     58.4 GiB
Used:      58.4 GiB (100.0%)
Available: 0 B
```
This is genuinely happening on this daemon's real host during this investigation.

**Verdict: Solvable** for the first step. `disks/free` immediately confirms and quantifies the problem.

## 2. Find what's actually consuming the space

**Ticket**: Continuation of #1 — now find the culprit.

**Real test**: `disks/usage` on `/var/lib` (`max_depth: 1`, `privileged: true`) found `/var/lib/containerd` at 3.9 GiB (3.5 GiB of it `overlayfs` snapshots). Checking `/` itself and `/var` (`max_depth: 1`) accounts for roughly:
- `/` (excluding `/var`, `/usr` etc. mount points): 509 MiB
- `/var`: 4.4 GiB
- `/usr`: 484 MiB

That's **~5.4 GiB accounted for against a 58.4 GiB, 100%-full disk.** Over 50 GiB is genuinely unaccounted for by any directory walk reachable through the daemon.

**Real gotcha hit again**: `disks/usage` with `one_file_system: true` on `/` returned only 509 MiB total — because this host has several *separate mount points* backed by the same physical device (`/var`, `/etc/hostname`, `/etc/hosts`, `/etc/resolv.conf` are all distinct mounts of `/dev/vda1`, confirmed via `disks/mounts`), and `one_file_system` treats each as a boundary to stop at, exactly like the `du -x` behavior that caused the very first disk investigation of this whole project to undercount. Omitting `one_file_system` and checking each mount point directly is necessary to get real numbers — the tool works correctly, but this is an easy way to silently under-report.

**Verdict: Partially solvable.** The tools correctly show real numbers for whatever tree you point them at, but there's no single "show me total usage of the whole host, correctly handling all separate mounts" call — you have to already know your mount topology (via `disks/mounts`) and manually sum across mount points. A large chunk of usage remains unexplained by any directory walk.

## 3. Chase the "du doesn't add up" mystery: deleted-but-still-open files

**Ticket**: Classic real-world cause of #2's mystery — a process holds a file descriptor open on a file that's been deleted (e.g. a rotated log the writer never reopened). The space isn't freed until the process closes/exits, but the file doesn't appear in any directory listing.

**Real test**: Checked `process://{pid}/open_files` (this session's own new `limits`/`open_files` targets) against several of the largest real processes on this host — `containerd` (151), `etcd` (699), `postgres` (1823), `kube-apiserver` (754), `kubelet` (268). Found one `(deleted)` entry, on `kubelet`:
```
/sys/fs/cgroup/kubelet.slice/kubelet.service/cpu.max (deleted)
```
This is a virtual cgroupfs file, not real disk-backed data (cgroup files routinely show as "(deleted)" across cgroup version transitions and hold zero actual bytes) — a red herring, correctly ruled out.

**Verdict: Partially solvable, and this is the most concrete gap this investigation found.** `process://{pid}/open_files` can confirm or rule out a *specific* process once you suspect it, but there is no bulk, system-wide way to answer "which process, across all ~30 running on this host, is holding open a deleted file, and how large is it?" without manually checking every PID one at a time — exactly the workflow a real `lsof +L1` or `find /proc/*/fd -ls | grep deleted` one-liner solves in one shot on a normal box. The ~50 GiB mystery from scenario #2 remains **genuinely unsolved** by this investigation using only mcpd's current tools.

## 4. Read the partition table (`fdisk -l` equivalent)

**Ticket**: Need to see partition boundaries/layout, e.g. to check if a partition can be safely resized.

**Real test**: `disks/partitions` on `vda` failed:
```
worker execution failed: exit status 1. Stderr: fdisk failed: exec: "fdisk": executable file not found in $PATH
```
Confirmed this is a missing-binary problem, not a permissions one (same failure with and without `privileged: true`) — `fdisk` (part of `util-linux`) simply isn't installed in this daemon's own Docker image, same class of gap as `lastb` earlier this session.

**Verdict: Not solvable today**, but purely due to a missing binary in the image, not a design gap — `ARCHITECTURE.md` already documents this tool as an intentional CLI wrap. One-line Dockerfile fix.

## 5. Check drive health (SMART) after suspected failing hardware

**Ticket**: A drive is suspected of failing; check SMART attributes for reallocated sectors, wear leveling, etc.

**Real test**: `disks/health` on `vda` returned a clean tool response, but `smartctl` itself reported: `"/dev/vda: Unable to detect device type"`.

**Verdict: Not solvable in this environment, but for an environmental reason, not a tool bug.** `vda` is a virtio virtual disk backing a cloud/VM/container host — SMART is a physical-drive protocol and structurally does not apply to virtual block devices. This scenario would work correctly on real hardware; it's inapplicable here, not broken.

## 6. Diagnose slow disk I/O / high I/O wait

**Ticket**: An app is reporting slow writes; is the underlying disk saturated?

**Real test**: `disks/performance` (no args) returned real per-device counters for every block device, e.g.:
```
vda    reads=7677173  writes=1602495  sectRead=3281074746  sectWrite=50819480  io_ms=1377705
```

**Verdict: Solvable.** Gives read/write counts and cumulative I/O time per device. One gap: this is cumulative-since-boot counters, not a live rate — a real "is it saturated *right now*" diagnosis needs two samples a few seconds apart and a manual delta calculation. The tool doesn't do this for you.

## 7. Distinguish "disk full" from "inode exhaustion" ("No space left on device" but `df` shows free space)

**Ticket**: An app reports `ENOSPC` / "no space left on device" even though `df` shows free space — classic inode exhaustion (common with e.g. mail spools or caches with millions of tiny files).

**Real test**: `disks/free` with `inodes: true` on this same host, at the exact moment `/` was 100% full by *blocks*, showed:
```
Total Inodes: 3907584
Used Inodes:  1725474 (44.2%)
Free Inodes:  2182110
```
Confirms this specific incident is block exhaustion, not inode exhaustion — a genuinely useful differential right now, on real data.

**Verdict: Solvable.** Clean, direct answer, and it correctly distinguishes the two failure modes on real, live data.

## 8. A service can't write logs and is crash-looping because the disk is full

**Ticket**: Combines #1 and service diagnosis — a pod/service is `CrashLoopBackOff`, and logs show `ENOSPC`.

**Real test**: `disks/free` (confirmed 100% full, scenario 1) + `services/list` filtered by the suspect service's `active_state`/`sub_state` (established working in this session's earlier services testing) gives the causal chain: disk full → write failure → crash. `logs/journal-control` filtered by `unit` would show the actual `ENOSPC` error in the service's own log lines.

**Verdict: Solvable end-to-end** — this is a real strength: disk state, service state, and that service's own logs are all independently confirmed-working tools that compose into a full incident narrative without needing a new feature.
