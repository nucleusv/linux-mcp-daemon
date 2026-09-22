# Users & Auth Troubleshooting

## 1. "User reports 'permission denied' writing to a directory they think they own"

**Ticket**: Classic support ticket — a user can't write somewhere they expect to.

**Real test**: `files/list` on `/root` **without** `privileged: true` failed cleanly: `failed to read directory: open /root: permission denied`. The exact same call **with** `privileged: true` succeeded and showed real content (`.bashrc`, `.kube/`, `.profile`, `.ssh/`). This is the real mechanism an admin would use: confirm the permission error exists, then use `privileged: true` (if authorized) to inspect as root and check actual ownership/mode.

**Verdict: Solvable** for the "confirm and work around" half. There's no dedicated `stat`-style "show me owner/group/mode for this path" - `files/list` shows names/sizes/mtimes but not ownership or permission bits in its output (confirmed from real output above: no UID/GID/mode fields shown) - so the actual root-cause detail ("it's owned by root:root, not this user") isn't directly visible without also reading the file's raw stat info.

## 2. List real users on the box (onboarding/access review)

**Ticket**: "Who has an account on this server?"

**Real test**: `users/list` with `min_uid: 1000` (via direct JSON-RPC, since this session's tool cache hadn't loaded the newly-shipped tool - see this investigation's README) returned, on the container: `ubuntu` (1000), `testuser` (1001), `unpriviliged` (1002), `privileged` (1003), `nobody` (65534) - real accounts, matching exactly the `useradd` calls in this project's own `Dockerfile`. Group memberships for `ubuntu` correctly show `sudo`, `adm`, `dialout`, etc.

**Verdict: Solvable.**

## 2b. Same check against the real host, not the container

**Real test**: (Established earlier this session, re-confirmed): the same call with `privileged: true` against the real host returns only `nobody` (65534) for `min_uid: 1000` — this specific kind node genuinely has no interactively-provisioned human users, consistent with having no SSH/login mechanism at all (see `network.md` scenario 3).

**Verdict: Solvable**, and correctly, honestly reflects "there's nobody to review" rather than erroring.

## 3. Check SSH key-based access for a user

**Ticket**: A deploy key stopped working; is it actually present in `authorized_keys`?

**Real test**: `files/find` on `/root/.ssh` for `authorized_keys` (with `privileged: true`) returned an empty result - no such file exists in this container's `/root/.ssh` (it has a `.ssh` directory, per scenario 1's listing, but no key file in it - unsurprising, since nothing SSHes into this specific worker container).

**Verdict: Solvable as a workaround** (generic `files/find` + `files/read` can locate and read any `authorized_keys` file that does exist), but there's no dedicated tool that understands SSH key formats/fingerprints/comments - just raw file access. Matches `plan/linux-admin-roadmap.md`'s already-tracked `auth/ssh-keys` gap.

## 4. Password/credential-related account checks (e.g. "is this account locked out?")

**Ticket**: A login is failing; is the account disabled or locked, not just a wrong password?

**Real test**: Not tested live - `users/list` deliberately never reads `/etc/shadow` (by design, documented in its own README as a security boundary), so account lock status, password age/expiry, and last-changed data are structurally out of scope for this tool.

**Verdict: Not solvable, by design.** This is a genuine gap for a specific diagnostic question, but it's a *deliberate* one (this daemon should not be reading password hash metadata), not an oversight - any future work here needs its own explicit, carefully-scoped design (e.g. exposing only lock-status/expiry booleans from `/etc/shadow`'s non-hash fields, never the hash itself), not a blanket "add shadow support."

## 5. Failed login / brute-force detection

**Ticket**: Suspicious repeated login failures; is someone trying to brute-force this box?

**Real test**: `logs/logins` with `type: "failed"` wraps `lastb` - already confirmed working mechanically elsewhere this session (though this specific host has neither the `wtmp`/`btmp` files nor even the `last`/`lastb` binaries on the real host, so no live data to show here - see this daemon's own earlier finding that this environment has no login mechanism at all).

**Verdict: Solvable as a tool, structurally inapplicable in this specific environment** - the same "tool works, environment has nothing to show" distinction as SSH/login scenarios elsewhere in this investigation.
