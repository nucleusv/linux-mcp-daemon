# FR-014 docker/run - launch containers from allowed images, with per-image allowed settings

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

### Decision (2026-09-29, owner) - supersedes the named-specs decision above

The owner's design, replacing named specs and the `containers:` name list:

- **The main gate is the image.** The grant lists image patterns (`*` matches within one path segment, `**` crosses `/`; references are normalised first, so `nginx` is `docker.io/library/nginx:latest`). A pattern that is only `*` is refused at load unless `allow_any_image: true`.
- **Each image pattern carries the settings the agent may use** (default-deny: a setting that is not listed cannot be used):
  ```yaml
  docker/run:
    allowed: true
    images:
      "ghcr.io/nucleusv/*":
        settings:
          memory: {max: 512m}                             # mandatory, up to this
          cpus:   {max: 1}
          ports:  {bind: 127.0.0.1, range: 18000-18999}   # loopback only, this port range
          env:    true
          command: true
          network: [bridge]                               # allowed networks; host is refused always
      "alpine:3.*":
        settings:
          memory: {max: 128m}
          network: [none]
          command: true
  ```
- **Refused for every image, whatever the grant says:** `--privileged`, capabilities, devices, security options, cgroup parent, sysctls, host or container namespaces (`pid`, `net`, `ipc`, `uts`, `userns`), `--env-file`, `--entrypoint`, `--health-cmd`, `--restart always`, and any mount of the Docker socket, `/`, `/etc`, `/root`, `/usr`. Mounts are only possible where a pattern lists them (path allowlist, read-only enforced, `..` and symlinks resolved).
- **No `containers:` list on `docker/run`.** The identity fence is automatic: every container it creates is named `mcpd-<mcpd user>-<name>` and labelled `mcpd.owner=<user>`, so no other user's `docker/manage`/`docker/exec` list can match it by accident; grant those tools on `mcpd-alice-*` where wanted.
- **`pull`:** default false (only images already on the host); allowing a pull is a per-image setting because a pull is arbitrary code arriving on the host.
- **Open, proposed defaults:** the agent may remove only containers labelled with its own `mcpd.owner` (through `docker/manage` scoped to `mcpd-<user>-*`); a per-grant `max_containers` quota counted by that label; `rm: true` allowed as a setting; a TTL is out of scope.

CLI (mock-up): `linuxctl run docker ghcr.io/nucleusv/linux-mcp-daemon:0.5.0 --memory 256m --publish 127.0.0.1:18080:9091`. Refusals name the pattern and the setting: `memory 2g exceeds the maximum 512m allowed for ghcr.io/nucleusv/*`; `image ... matches no pattern in your docker/run images:`.

## Acceptance criteria

- [ ] `docker/run` takes an image reference and the settings of the design above; nothing outside the listed settings reaches the Engine API - the `POST /containers/create` body is built by mcpd from the validated settings, never from raw caller fields.
- [ ] Image matching: normalisation, `*`/`**` semantics, `allow_any_image` refusal, digest patterns (`@sha256:*`), and a clear message listing the patterns when nothing matches.
- [ ] Per-image settings enforced default-deny with maxima (`memory`, `cpus`, `pids`), loopback-only port bindings within a range, allowed networks, mounts only from a listed read-only path allowlist; the always-refused list is refused at call time and at config load.
- [ ] Automatic name `mcpd-<user>-<name>` and labels `mcpd.owner`, `mcpd.image-pattern`; `max_containers` quota by owner label.
- [ ] `pull` false by default; when allowed for a pattern, audited.
- [ ] `docker/run` is audited at any log level with user, image, resolved name and the applied settings (no env values).
- [ ] The open defaults above settled by the owner and recorded.
- [ ] Docs: tool page with live output, `permissions-and-risks.md` (why images plus settings, what an image allowlist does not fence, tag mutability), `mcp-sudo.md` for the `images:` block, README examples, `linuxctl run` in the command reference and man page; `check_docs.sh`, `check_readmes.sh`.
- [ ] Deployed and checked live on local k8s (clear no-socket error), VPS 9091 and 9092, on throwaway images and names, never touching the VPN container on the host.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Image matching | Go unit tests: normalisation, `*` vs `**`, digest, `*` alone refused, no match message | as specified | | |
| T2 | Settings default-deny | Go unit tests: an unlisted setting, a value over a maximum, a non-loopback bind, a network not allowed | each refused with the pattern and setting named | | |
| T3 | Always-refused list | Go unit tests for every option in the list, at config load and at call time | refused, never forwarded | | |
| T4 | Body built by mcpd | Go unit test against a fake Engine: caller-supplied raw `HostConfig` keys absent from the request | none forwarded | | |
| T5 | Name and labels | Go unit test + live: `mcpd-<user>-<name>`, `mcpd.owner` present | as specified | | |
| T6 | Quota | Live, VPS 9091: `max_containers: 2`, run three times | third refused | | |
| T7 | Audit | Live: journal line per run at info level, no env values | as specified | | |
| T8 | No pull | Live: allowed pattern, image absent | refused "not present", no fetch | | |
| T9 | Cleanup | Live: `rm: true`, container gone after exit | as specified | | |
| T10 | Docs/build | `check_docs.sh`, `check_readmes.sh`, `GOOS=linux go build ./...`, `go test ./...` | pass | | |

## Comments

- 2026-09-27 - created: the owner asked what boundaries a `docker run` needs
  ("docker image path, registry, mounts, mapping, volumes, what else?"). The
  boundary list and the launch-spec decision above are the answer. Not started -
  FR-011, FR-013 and FR-012 come first, and this ticket only exists so the
  analysis is not lost in chat.
- 2026-09-29 - redesigned with the owner: direct run gated by allowed image patterns with per-image allowed settings, no `containers:` list on `docker/run`, automatic `mcpd-<user>-` names, default-deny settings. Part of release v0.5.0 (FR-028); built after FR-026.
