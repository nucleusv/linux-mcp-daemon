# Investigations

Empirical answer to: "if you had only `mcpd`, could you actually troubleshoot a real Linux system?" Every scenario in this folder was tested live against this project's actual deployed daemon (a real Kubernetes-in-Docker node on Docker Desktop for Mac) — not simulated, not hypothetical. Several scenarios (the disk hitting 100% full, `journalctl`/`fdisk`/`lastb` missing binaries, `processes/list`'s broken CPU sort, the `vstretchi` app not actually listening on its expected port) are real conditions this investigation found *while testing*, not staged examples.

**Methodology**: each category file lists realistic troubleshooting tickets (including several sourced from classic SRE/DevOps/platform-engineering interview questions — "the website is down, walk me through your diagnosis," "load average is high, what do you check," "a pod is CrashLoopBackOff, how do you debug it," "disk is full, what's your process" — noted inline where used, so the reasoning is auditable) and maps each to specific `mcpd` tool/resource calls, actually executed, with real output. Every scenario is marked **Solvable**, **Partially solvable**, or **Not solvable**, with the specific missing piece named for the latter two. Where the environment itself (Docker Desktop sandboxing, a kind node with no login mechanism, a virtual disk with no SMART support) rather than the tool explains a limitation, that's called out explicitly rather than counted as a tool defect.

No mutating/destructive calls were made — every test is read-only diagnosis, even for scenarios that would naturally end in a fix (e.g. restarting a service).

## Files

- [disk-storage.md](disk-storage.md) — including a live, still-unsolved "disk full but `du` doesn't add up" mystery
- [cpu-performance.md](cpu-performance.md) — including the confirmed-broken `sort_by: cpu`
- [memory.md](memory.md)
- [network.md](network.md) — including a live "process running but port not listening" real diagnosis
- [services-systemd.md](services-systemd.md) — including confirmation that systemd timers aren't visible at all
- [users-auth.md](users-auth.md)
- [packages-updates.md](packages-updates.md)
- [security-access.md](security-access.md)
- [logs-diagnostics.md](logs-diagnostics.md)
- [boot-kernel-hardware.md](boot-kernel-hardware.md)
- [containers-kubernetes.md](containers-kubernetes.md) — specific to this daemon's actual Kubernetes-node deployment context
- [feature-backlog.md](feature-backlog.md) — the synthesized result: every gap found, cross-referenced to the scenario that found it, each with a concrete post-implementation test plan

## A methodology note worth recording

This investigation's session had a stale MCP tool cache that hadn't been reconnected since several tools (`disks/mounts`, `users/list`, `logs/logins`) were added — the structured `mcp__linux-mcp-daemon-by-claude-priv__*` interface didn't know they existed. Rather than skip testing them, every scenario needing those tools was run via direct JSON-RPC (`curl` against the daemon's real `/sse`+`/message` endpoints) — the same authoritative interface the structured tools use underneath, and the same technique this project has used all session to verify claims against real behavior instead of assumptions. This is a client-side caching quirk of the MCP tooling, not something to fix in `mcpd` itself, but worth knowing if a future investigation hits the same "no matching deferred tools found" wall.
