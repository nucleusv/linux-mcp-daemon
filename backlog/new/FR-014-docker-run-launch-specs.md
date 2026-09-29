# FR-014 docker/run - launch containers from named specs

- **Created:** 2026-09-27, by the owner
- **Release:** [FR-028](../in-progress/FR-028-release-0.5.0.md) - v0.5.0
- **Related:** FR-011 (the docker group, container allowlists), FR-012 (prune), FR-013 (networks), `internal/docker/allow.go`, `configs/mcp-sudo.yaml`, `docs/website/docs/configuration/permissions-and-risks.md`

## Description

FR-011 deliberately ships no way to create a container: `docker/manage` starts and
stops what already exists. The owner asked what boundaries a `docker/run` would
need. The answer decides the shape of the tool, so it is recorded here before any
code.

`docker run` is not one boundary, it is ~10 independent paths to host root, and
one unfenced path makes the other nine pointless:

- **Instant host root:** `--privileged`, `--cap-add`, `--security-opt`
  (`seccomp=unconfined`, `apparmor=unconfined`), `--device`, `--runtime`,
  `--cgroup-parent`, `--sysctl`; a bind mount of `/`, `/etc`, `/root`, `/usr`, and
  above all `/var/run/docker.sock` (mount the socket, own the host); namespace
  sharing (`--pid=host`, `--net=host`, `--ipc=host`, `--uts=host`,
  `--userns=host`, `--net=container:<id>`, `--pid=container:<id>`);
  `--env-file` (mcpd reads that path *as root*); `--entrypoint` and
  `--health-cmd` (arbitrary code, the second one on a timer).
- **Needs a positive allowlist:** image registry + repo + tag/digest; pull policy
  (a pull is arbitrary code landing on the host - "never pull, images already
  present only" is the cheap default); bind-mount paths (with `..`/symlink
  resolution and `ro` enforcement); named volumes; host ports *and* bind address
  (default `127.0.0.1`, never `0.0.0.0`); network names, with `host` and
  `container:` refused outright.
- **Needs a floor or it is a host DoS:** mandatory `memory`, `cpus`,
  `pids-limit` with a per-grant maximum; `--restart=always` is a foothold that
  survives reboot.
- **Identity fence (the one that gets missed):** the new container's name must
  fall inside *this user's own* `containers:` glob. Otherwise creation is a
  lateral-movement primitive - create `web-1`, and whoever holds `web-*` on
  `docker/exec` now has a container this user controls. Force an
  `mcpd.owner=<user>` label for audit and cleanup, plus a per-user container
  quota and a TTL.

**Decision (2026-09-27):** do not validate the Engine API field by field.
`POST /containers/create`'s `HostConfig` has ~60 fields and grows with every
Docker release: a denylist is never finished, and a whitelist over all of it is a
project of its own. Instead the grant defines **named launch specs** and the call
picks one, filling at most a fenced blank:

```yaml
docker/run:
  allowed: true
  specs:
    debug-shell:
      image: alpine:3.22          # exact ref, never pulled
      command: ["sleep", "3600"]  # fixed
      name: fr011-debug-*         # must also match containers:
      memory: 256m
      cpus: 0.5
      network: none
      rm: true
```

The caller passes `spec: debug-shell` and optionally a name suffix. No mounts,
ports or capabilities to validate, because none are expressible. Free-form run
stays out until there is a case the specs cannot serve.

## Acceptance criteria

- [ ] `docker/run` takes `spec` (required) and `name_suffix` (optional) and
      **nothing else** - no image, no mounts, no ports, no caps in the call.
- [ ] The container body sent to `POST /containers/create` is built from the spec
      in mcp-sudo.yaml only; no field of the caller's arguments reaches `HostConfig`.
- [ ] Spec validation is strict at config load: unknown keys refused, `image`
      required, `memory` and `cpus` required, `network: host` refused, any bind
      mount refused unless its path is inside a `paths:`-style allowlist in the
      same spec, `/var/run/docker.sock` refused unconditionally.
- [ ] The resolved name must match the user's `docker/run` `containers:` glob, or
      the call is refused before the socket is dialed.
- [ ] Every created container carries `mcpd.owner=<user>` and `mcpd.spec=<spec>`.
- [ ] A per-grant `max_containers` caps how many `mcpd.owner=<user>` containers
      may exist; the call is refused at the cap.
- [ ] `docker/run` is in `mutatingTools`, so it is audited at any log level, with
      the spec name and the resolved container name in the audit line.
- [ ] Docs: tool page with live output, a `permissions-and-risks.md` section on
      why the spec list exists rather than free-form run, `mcp-sudo.md` on the
      `specs:` block.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Spec resolution: `spec: debug-shell` creates the container the spec describes | Go unit test against a fake Engine | Request body matches the spec; caller args absent | | |
| T2 | Caller cannot inject `HostConfig` | Go unit test: call with `binds`, `privileged`, `network_mode` in args | Those keys are dropped, not forwarded | | |
| T3 | Unknown spec name | Go unit test | Refused, lists the spec names available to this user | | |
| T4 | Strict config check | `internal/config` unit test: spec with `network: host`, one with a `/` bind, one mounting the docker socket, one without `memory` | Each refused at load with a message naming the key | | |
| T5 | Name fence | Go unit test: spec whose `name` falls outside `containers:` | Refused before dialing the socket | | |
| T6 | Quota | Live, VPS 9091: `max_containers: 2`, run three times | Third call refused; two containers exist | | |
| T7 | Labels | Live, VPS 9091 | `mcpd.owner`/`mcpd.spec` present on the created container (`docker/containers -o json`) | | |
| T8 | Audit at info level | Live, VPS 9091 journal | One audit line per run with user, spec, resolved name; no spec internals leaked | | |
| T9 | No pull | Live, VPS 9091: spec naming an absent image | Refused with "image not present", no network fetch | | |
| T10 | Cleanup path | Live, VPS 9091: `rm: true` spec | Container gone after exit; `docker/volumes` unchanged | | |

## Comments

- 2026-09-27 - created: the owner asked what boundaries a `docker run` needs
  ("docker image path, registry, mounts, mapping, volumes, what else?"). The
  boundary list and the launch-spec decision above are the answer. Not started -
  FR-011, FR-013 and FR-012 come first, and this ticket only exists so the
  analysis is not lost in chat.
