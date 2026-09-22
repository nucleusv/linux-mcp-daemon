# Packages & Updates Troubleshooting

## 1. "What's installed on this box?" (baseline inventory, incident-response first step)

**Ticket**: Establishing a baseline, or checking whether a specific package/version is present after a suspected compromise or bad deploy.

**Real test**: `system/packages` with `privileged: true` returned the real host's full package list (~190 packages) - e.g. `iptables 1.8.11-2`, `nftables 1.1.3-1`, `systemd 257.13-1~deb13u1`, `tzdata 2026b-0+deb13u1`. Confirms the earlier `system/packages`-vs-container distinction from earlier this session still holds, and additionally confirms **the real host does have both `iptables` and `nftables` installed** - directly relevant to `plan/linux-admin-roadmap.md` item 3 (firewall visibility): a future `network/firewall-rules` read tool would have real data to query on this host, this isn't a dead end.

**Verdict: Solvable.**

## 2. Check for available security updates

**Ticket**: "Are there pending patches, especially security ones, we haven't applied?"

**Real test**: No tool exists for this - `system/packages` only lists *installed* packages, nothing about what's available upstream (`apt list --upgradable` equivalent). Confirmed absent from the schema.

**Verdict: Not solvable.** Matches `plan/linux-admin-roadmap.md` item 7's already-tracked `security/updates-available` gap - this investigation didn't find anything to add beyond confirming it's real and unaddressed.

## 3. Install/remove/upgrade a package (incident remediation)

**Ticket**: A fix requires installing a package, or removing a compromised/vulnerable one.

**Real test**: Not tested (mutating action, out of scope for this investigation's methodology) - confirmed absent from the schema regardless.

**Verdict: Not solvable.** Matches roadmap item 4.

## 4. "A dependency is missing / a binary isn't found" - this investigation hit this exact scenario repeatedly, for real

**Ticket**: This isn't hypothetical - during this investigation, `disks/partitions` (`fdisk`), `logs/logins` type `"failed"` (`lastb`, on the real host), and `logs/journal-control` (`journalctl`, without `privileged: true`) all failed with `executable file not found in $PATH`.

**Real test**: `system/packages` correctly lists what *is* installed, letting a caller confirm a suspected-missing binary's package is genuinely absent rather than guessing - e.g. confirming this daemon's own container image lacks a `fdisk`-providing package.

**Verdict: Solvable as a confirmation step**, but there's no tool that directly answers "which package provides binary X" (a `dpkg -S`/`which`-plus-package-lookup equivalent) - today an admin has to separately try running the binary (and get the raw exec error) and cross-reference the package list by hand.
