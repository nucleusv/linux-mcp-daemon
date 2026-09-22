# Services & systemd Troubleshooting

## 1. "A service was just restarted but users report it's still down"

**Ticket**: Ops restarted a service; is it actually healthy now?

**Real test**: `services/list` filtered to `pattern: "*kubelet*"` returned real live state:
```
[active] kubelet.service
  State: running (active) | Load: loaded
```
Combined with `service://{name}/status` (confirmed working elsewhere this session, shows `fragment_path` etc.) and `logs/journal-control` (see scenario 3), gives ActiveState/LoadState/SubState plus the unit's own recent log lines.

**Verdict: Solvable.**

## 2. "A pod is `CrashLoopBackOff`, how do you debug it?" (classic platform-engineering interview question)

**Ticket**: A workload keeps restarting.

**Real test**: `services/list` with `active_state: "failed"` returned `No services found matching the criteria` (correctly confirming zero failed units right now — a clean negative result, not an error). The combination that *would* diagnose an actual failure: `services/list` (confirm `failed`/`activating` state) → `logs/journal-control` filtered by `unit` (get its last log lines before death) → `processes/list` (check if a zombie/orphan process remains).

**Verdict: Solvable as a workflow** (each piece independently confirmed working elsewhere in this investigation), though this is Kubernetes-pod-level crash-looping (`kubectl describe pod`, container exit codes, `Back-off restarting failed container` events) — this daemon's tools operate at the **host systemd / process** level, not the Kubernetes API level, so a real `CrashLoopBackOff` investigation on this specific host would need `containers/list`-style tooling (see `plan/linux-admin-roadmap.md` item 8) to see it from the container-runtime side, not just the host-process side.

## 3. View a service's own logs to find why it crashed

**Ticket**: Continuation of #1/#2 — read the actual error.

**Real test**: `logs/journal-control` filtered to `unit: "kubelet.service"` **failed without `privileged: true`**: `journalctl: executable file not found in $PATH` — because this daemon's own container image doesn't bundle `journalctl` at all. With `privileged: true`, it succeeded and returned real, live log lines (`containerd[151]: ... container event discarded ...`).

**Verdict: Solvable, but with an important caveat worth documenting explicitly**: unlike most tools where `privileged: true` is optional (needed only for permission reasons), `logs/journal-control` is **unusable at all** without it in this deployment, since the binary itself only exists on the real host, not in the container. A caller who doesn't know this will get a confusing "executable not found" error that looks like a broken tool rather than "you forgot a required flag."

## 4. List systemd timers (cron-equivalent visibility) — resolves an open question from the roadmap

**Ticket**: `plan/linux-admin-roadmap.md` item 2 flagged this as worth verifying before assuming a gap: "`services/list` may already surface `.timer` units generically... worth verifying."

**Real test**: `services/list` with **no filter at all** (`privileged: true`) returned 56 real units — every single one a `.service` unit. Zero `.timer`, `.socket`, `.target`, or `.mount` units appeared, even though a normal systemd host always has several loaded (`sockets.target`, `dbus.socket`, etc.). This strongly confirms the underlying `go-systemd/v22/dbus` query is scoped to services only.

**Verdict: Not solvable — now confirmed, not assumed.** This resolves the roadmap's open question with real evidence: `services/list` does **not** surface timers. Scheduled-task visibility via systemd timers remains a genuine, now-verified gap, not just a documentation gap.

## 5. Check for any failed units system-wide (routine health check)

**Ticket**: "Is anything broken on this box right now?" — the systemd equivalent of `systemctl --failed`.

**Real test**: `services/list` with `active_state: "failed"` returned `No services found matching the criteria` — a real, clean, live answer (nothing is failed on this host right now).

**Verdict: Solvable**, and this is a good one-call health check.

## 6. Enable/disable a service across reboots

**Ticket**: A service should (or shouldn't) start automatically on boot.

**Real test**: `services/manage`'s schema accepts `action: "enable"`/`"disable"` (confirmed in the tool's own inputSchema, not executed here to avoid a mutating test action per this investigation's methodology).

**Verdict: Solvable** (not executed, but the capability is real and already shipped, not a gap).
