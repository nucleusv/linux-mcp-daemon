# Network Troubleshooting

## 1. "The website is down, walk me through your diagnosis" (classic SRE interview question)

**Ticket**: A service is unreachable; find out why.

**Real test**: `network/connections` (no filters, `privileged: true`) returned every real listening socket on this host with **process attribution included**, e.g.:
```
tcp LISTEN *:6443 *:* users:(("kube-apiserver",pid=754,fd=4))
tcp LISTEN *:9091 *:* users:(("mcpd",pid=383250,fd=4))
```

**Verdict: Solvable**, and better than expected going into this investigation — process-to-port correlation ("which process owns this port") already works natively, contrary to what might be assumed as a gap. This is exactly the first real diagnostic step for "is anything even listening."

## 2. A real, live example: "the process is running but the port isn't responding"

**Ticket**: This is not hypothetical — it's happening on this host right now. A `vstretchi` app (`sh -c "alembic upgrade head && uvicorn vstretchi.main:app --host 0.0.0.0 --port 8000"`) has a running `uvicorn` process (PID 303913, alive, 92 MB RSS, state `S`), but:

**Real test**: `network/connections` filtered to `port: 8000` returned **zero rows** — confirmed the filter mechanism itself works correctly by cross-checking `port: 6443` (a known-good, confirmed-listening port), which correctly returned exactly the `kube-apiserver` line. So port 8000 genuinely has nothing listening on it. `processes/list` filtered by `user: root` (full listing) shows a `pg_isready -U vstretchi` health-check process (PID 387234) **currently running** at the same time, alongside a real `postgres` process — strongly suggesting `uvicorn`'s startup chain is stalled waiting on a Postgres dependency and never reached the point of binding its socket, or bound and then stopped listening.

**Verdict: Solvable as a diagnosis** — the combination of `network/connections` (proves nothing is listening) + `processes/list` (proves the process still exists) + spotting the concurrent `pg_isready` health-check is exactly the real evidence trail a human would use to conclude "this app is stuck waiting on its database, not actually crashed." No new tooling needed; this is a genuine success story for the existing toolset, caught live rather than staged.

## 3. Can't SSH in — is sshd even listening?

**Ticket**: SSH connection refused/times out.

**Real test**: Same `network/connections` call as scenario 1 — no `sshd` process/port appears in the real listing on this host (expected: this is a minimal kind node with no SSH access model at all, consistent with the earlier finding in this session that it has no `last`/`lastb` binaries or `wtmp` either).

**Verdict: Solvable as a tool** (the exact same call answers "is sshd listening" on any host that has one), **structurally inapplicable in this specific environment** — not a gap, an environment property.

## 4. DNS resolution failure — external vs. cluster-internal (real, both tested)

**Ticket**: An app can't resolve a hostname.

**Real test, external DNS**: `network/nslookup` for `google.com` returned a full, real, correct response (A/AAAA/MX/NS/TXT records) — works cleanly.

**Real test, cluster-internal DNS**: `network/nslookup` for `kubernetes.default.svc.cluster.local` (a real in-cluster service name, referenced directly in this node's own `kube-apiserver` cmdline as `--service-account-issuer`) **timed out after the full 30-second worker timeout** rather than returning NXDOMAIN or any error.

**Verdict: Partially solvable, with a real structural explanation, not a bug.** This daemon's pod runs with `hostNetwork: true` (documented elsewhere in this repo), so its DNS resolution goes through the **host's** `/etc/resolv.conf`, not the cluster's pod-network CoreDNS resolver — it structurally cannot resolve `*.svc.cluster.local` names, and instead of failing fast it hangs for the full worker timeout, which is a worse failure mode than a quick NXDOMAIN (a 30-second hang looks like the tool is broken, not like a resolver limitation). Worth fixing: either a faster timeout specifically for DNS, or documentation on this daemon's specific inability to resolve cluster-internal names when run with `hostNetwork: true`.

## 5. Is a firewall rule blocking this traffic?

**Ticket**: A connection is refused/dropped and a security group or `iptables` rule is suspected.

**Real test**: No tool exists for this at all — confirmed absent from the schema (matches `plan/linux-admin-roadmap.md` item 3, already flagged there).

**Verdict: Not solvable.** Real, known, already-tracked gap.

## 6. "A pod is `CrashLoopBackOff`, how do you debug it?" (classic SRE/platform-engineering interview question, network angle)

**Ticket**: Same classic question as elsewhere in this investigation, focused on the "is it even reachable" half.

**Real test**: Chaining `services/list` (confirmed working elsewhere this session) with `network/connections` (scenario 1) and `processes/list` (scenario 2) gives the full picture: is the service's process alive, is it listening, and — via `logs/journal-control` filtered by `unit` — what does it log right before dying. All three tools independently confirmed working; composing them answers this question end to end without new tooling, same conclusion as `disk-storage.md` scenario 8.

## 7. Network path / latency to a remote host

**Ticket**: High latency or intermittent packet loss to an external service.

**Real test**: `network/trace-path` to `8.8.8.8` returned the real first hop (`172.19.0.1`, the Docker network gateway, 0.05–0.46ms) then `* * *` for hops 2 through 5.

**Verdict: Solvable as a tool, environment-limited result.** The tool ran correctly and reported exactly what it saw; Docker Desktop's networking sandbox does not forward/respond to further traceroute probes past its own gateway, so deeper path tracing isn't possible from this specific environment — an environment property, not a tool defect (same category as `disks/health`'s SMART-on-virtual-disk limitation).

## 8. Local network / ARP resolution issue

**Ticket**: A host on the local subnet isn't reachable; is ARP resolving its MAC address?

**Real test**: `network/arp` returned the full, real ARP cache — every pod veth pair and Docker bridge peer on this node, with real MAC addresses.

**Verdict: Solvable.**

## 9. Check network interface configuration and traffic counters

**Ticket**: Confirm an interface's IP/MTU config and check for RX/TX errors.

**Real test**: `network://interfaces` resource returned full real data per interface (`lo`, `tunl0`, `gre0`, `gretap0`, ...) including addresses, MTU, flags, and RX/TX byte/packet/error/drop counters.

**Verdict: Solvable.**
