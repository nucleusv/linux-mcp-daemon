# docker/manage

Container lifecycle over the Engine API: `start`, `stop`, `restart`, `kill`, `pause`, `unpause`, `remove` — what `services/manage` is for systemd units.

`remove` is the one irreversible action and is deliberately fenced: the call never sends Docker's `force` or `v`, so a running container is refused (stop or kill it first — two separate, separately audited calls) and anonymous volumes are never deleted.

An action whose state the container is already in (a `start` on a running container) comes back from Docker as `304` and is reported as a no-op — `result: "unchanged"` in structured output — not as an error.

## Parameters
- `container` (string, required): name, full ID or ID prefix.
- `action` (string, required): `start`, `stop`, `restart`, `kill`, `pause`, `unpause`, `remove`.
- `output_format` (string, optional): `json`, `yaml`; default text.

## Usage & Permissions
⚠ Root or nothing, and it changes the host: needs `docker/manage: {allowed: true, containers: [...]}` in the caller's grant in `configs/mcp-sudo.yaml`. A grant with no `containers:` list refuses every container rather than allowing all of them; `["*"]` allows all.

The identifier is **resolved first, then checked**: the daemon matches the canonical name *and* the ID against the allowlist and acts on the ID, so a `docker rename` between check and action cannot move the call to another container. An identifier that does not match is rejected, never sanitized.

`linuxctl stop docker <name>` and friends call this tool (the verb comes from the `action` enum).
