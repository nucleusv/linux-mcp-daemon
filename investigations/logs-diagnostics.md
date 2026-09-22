# Logs & Diagnostics

## 1. Read the kernel ring buffer for hardware/driver errors

**Ticket**: Suspected hardware issue, or need to see boot-time kernel messages.

**Real test**: `logs/dmesg` filtered to `level: "err,warn"` (`privileged: true`) returned real, live kernel messages from this host - including a security-relevant finding reused in `cpu-performance.md`: `cfs_period_timer[cpuN]: period too short, scaling up` (cgroup CPU-quota pressure) and repeated `ICMPv6: NA: ... advertised our address` warnings (duplicate address detection noise, typical in a busy container/pod network).

**Verdict: Solvable.**

## 2. Query the systemd journal, filtered by unit/time

**Ticket**: The general-purpose "show me what happened" tool for any service or time window.

**Real test**: `logs/journal-control` filtered by `unit: "kubelet.service"` and `lines: 5` - failed without `privileged: true` (`journalctl` binary missing from this daemon's own container image), succeeded with it, returning real live log entries. Fully documented in `services-systemd.md` scenario 3 - repeated here because this is arguably the single most load-bearing diagnostic tool in the entire toolset, used to close out nearly every other category's "why did it fail" question.

**Verdict: Solvable, with the important caveat that `privileged: true` is mandatory here, not optional**, in this specific containerized deployment.

## 3. Correlate a timestamp across multiple log sources

**Ticket**: An incident happened at a specific time; pull kernel, journal, and service-state data for that exact window to build a timeline.

**Real test**: `logs/dmesg` has no `since`/`until` time filter (only `level`) - it returns whatever the kernel ring buffer currently holds, oldest-relative-timestamps included in the text but not filterable. `logs/journal-control` *does* support `since`/`until`. So time-windowed correlation works for the journal but not for dmesg.

**Verdict: Partially solvable** - one of the two log sources supports time-range filtering, the other doesn't, making a unified "everything that happened between 14:02 and 14:05" query impossible in one step; an admin would need to fetch all of `dmesg` and manually scan for the relevant timestamps.

## 4. "System time / logs look wrong" (timezone-related log confusion)

**Ticket**: Log timestamps don't match wall-clock expectations; is the system's timezone misconfigured?

**Real test**: `system://timezone` (shipped this session) — real test on the container found **no `/etc/timezone` and no `/etc/localtime`** at all (this base image has no `tzdata` configured), correctly falling back to reporting `UTC` per documented POSIX/glibc default behavior rather than erroring or guessing. Cross-checked against `system/packages`: the **real host** *does* have `tzdata` installed (`packages-updates.md` scenario 1) — meaning the host and container likely have genuinely different effective timezones, and `system://timezone` (being read directly in the master daemon process, not worker-routed) can only ever report the **container's**, never the real host's, even with `worker.containerized: true`.

**Verdict: Solvable for "what does this container think the time is," not solvable for "what does the real host think the time is."** This is a concrete, real consequence of the architectural note already in `system://hostname`'s own README (direct-read resources never see the real host) - worth highlighting here because it directly affects a log-timestamp-correlation scenario, not just an abstract config value.
