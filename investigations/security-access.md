# Security & Access Troubleshooting

## 1. Find SUID/SGID binaries (routine security audit)

**Ticket**: Security review requires listing all SUID/SGID binaries system-wide, to check for unauthorized privilege-escalation vectors.

**Real test**: `files/find`'s schema (confirmed by direct inspection: `path`, `name`, `type`, `mtime`, `size`, `max_depth`, `privileged`) has **no permission/mode filter parameter at all** - there's no way to ask "find files with the setuid bit set" through this tool.

**Verdict: Not solvable.** Matches `plan/linux-admin-roadmap.md` item 7's `security/suid-scan` gap, now confirmed by direct schema inspection rather than assumption - `files/find` cannot be worked around for this via existing parameters.

## 2. Confirm exactly what a specific authenticated caller is allowed to do

**Ticket**: "Why did my call get rejected?" / a real self-service permissions check, or debugging an authorization config change.

**Real test**: `auth/sudo-rules` returned this session's real, complete authorization set for the calling user (`testuser`): every `Tools` entry with `Allowed`/`Paths`, every `Resources` entry with its granted prefixes. Genuinely useful, real output - e.g. it correctly shows `files/list`/`files/create`/`files/update`/`files/filetype` all have `Paths: ["/"]` but critically **`files/find` and `files/read` are absent from the list entirely** - which explains, before you even try, why `files/find` calls fail for this user (confirmed failing live in `disk-storage.md`/`users-auth.md`).

**Verdict: Solvable**, and a genuinely strong tool - this is exactly the kind of self-diagnosing capability that prevents "why doesn't this work" confusion, and it caught a real gap (testuser's incomplete `files/*` grants) just by reading it.

## 3. Firewall rule inspection

**Ticket**: Confirm whether a specific port/IP is blocked at the firewall level, not just unreachable at the application level.

**Real test**: No tool exists. `system/packages` (see `packages-updates.md` scenario 1) confirms `iptables`/`nftables` are genuinely installed on the real host, so there would be real rules to read if a tool existed.

**Verdict: Not solvable.** Matches roadmap item 3, now with confirmed evidence that the underlying binaries exist on this host and a read tool would have real data to show.

## 4. Detect unauthorized/unexpected privileged access attempts

**Ticket**: Did anyone try to escalate privileges or access something they shouldn't have?

**Real test**: `logs/journal-control` (confirmed working with `privileged: true`, see `services-systemd.md` scenario 3) can filter by `unit`, `since`, `until` - this would surface `sudo`/`polkit`/`PAM` denial messages if filtered toward the right unit/timeframe, but there's no dedicated "show me auth failures" filter; it's the same generic journal search used for every other log-based scenario in this investigation, not a security-specific view.

**Verdict: Partially solvable** - the raw capability (search the journal) works, but there's no purpose-built "security events" lens on top of it. Matches roadmap item 7's `security/failed-logins`.

## 5. Verify privilege boundaries are actually enforced (not just configured)

**Ticket**: Confirm that an unprivileged caller genuinely cannot access root-gated resources - a security *test*, not just a diagnostic.

**Real test**: Already empirically confirmed working correctly earlier this session (documented in `ARCHITECTURE.md`'s test coverage): the `unpriviliged` user is correctly denied `devices://usb` with a clear `-32603` error rather than silently succeeding. Not re-tested destructively here, but this is a real, previously-verified security property of the daemon itself, worth noting as a genuine strength rather than a gap.

**Verdict: Solvable**, and already proven correct, not just assumed.
