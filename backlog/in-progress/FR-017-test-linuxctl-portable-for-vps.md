# FR-017 tests/test_linuxctl.sh runs on the VPS stands, not only on local k8s

- **Created:** 2026-09-29, by the owner ("test_linuxctl.sh rewrite for vps")
- **Related:** `tests/test_linuxctl.sh`, `tests/run_all.sh`, FR-015 / FR-016 (docker templates, extra-word warning, completion)

## Description

`tests/test_linuxctl.sh` only runs against the local kind/Docker Desktop node: it hardcodes the disk `vda`, the interface `eth0`, the service `kubelet.service`, the pattern `kube*`, the token values, `http://localhost:9091`, builds `linuxctl` with `go build`, and marks `logs/logins` as an expected failure (that node has no `last`). None of that holds on the VPS stands (89.125.210.117, 9091 systemd and 9092 Docker), which are TLS, have a real login history, run `ssh.service`, and have their own tokens. It also has no docker checks at all.

Rewrite it so one script serves every target: everything environment-specific comes from env vars with the current local-k8s values as defaults (so `run_all.sh` locally keeps working unchanged), the disk, interface and service are discovered where they can be, an already-built `linuxctl` can be used, and a docker section (group visible → reads, the FR-015 template forms, the typo error, the FR-016 warning) runs only when the group is granted.

## Acceptance criteria

- [ ] All target-specific values are env-configurable with local-k8s defaults; nothing in the script needs editing to point it at a VPS stand.
- [ ] Disk device and network interface are discovered from the target itself, not hardcoded.
- [ ] `LINUXCTL=/path` skips the `go build`; TLS works through the usual `MCP_CA_CERT` / `MCP_TLS_FINGERPRINT` env.
- [ ] `logs/logins` is required to pass where `LOGINS_MAY_FAIL=no`, tolerated only where set to `yes` (the local default).
- [ ] A docker section runs when `docker/containers` is granted to the privileged user: bare list forms, `get docker network bridge`, `get docker networkz` must exit non-zero, `get docker network bridge extra-word` must print the extra-argument warning; skipped with a visible message otherwise.
- [ ] The script exits non-zero on the first real failure, prints a final pass line only when everything passed.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Syntax | `bash -n tests/test_linuxctl.sh` | clean | 2026-09-29 | pass - `bash -n` clean |
| T2 | VPS Docker stand | run on the VPS against 9092 with the testuser/privileged/unpriviliged tokens, `LOGINS_MAY_FAIL=no` | exit 0, docker section runs | 2026-09-29 | pass - VPS 9092 (89.125.210.117) as testuser/privileged/unpriviliged, `LOGINS_MAY_FAIL=no`, `DOCKER_EXEC_CONTAINER=fr011-probe`: exit 0, 183 steps, 10,414 lines, docker section ran (bridge template, `networkz` error, extra-word warning, `exec -- echo hello`, ungranted refusal) |
| T3 | VPS systemd stand | same against 9091 (needs three users at the three privilege levels there) | exit 0 | 2026-09-29 | not run - 9091 has only mcp/agent/fr011, no three privilege levels; needs the owner to create them |
| T4 | Local k8s, defaults | `bash tests/run_all.sh` / the script with no env | same behaviour as before the rewrite, exit 0 | 2026-09-29 | pass - local k8s with no env: exit 0, 164 steps, `logs/logins` and devices://dmi tolerated as before, docker section skipped (no docker socket on the node), the built ./linuxctl removed by the exit trap |
| T5 | Negative: bad token | run with a wrong `TOKEN` | fails fast with a clear message, exit non-zero | 2026-09-29 | pass - `TOKEN=wrong-token`: `❌ FAILED: linuxctl failed to ping the daemon at http://localhost:9091.`, exit 1 |

## Comments

- 2026-09-29 - created and moved straight to in-progress: the owner asked for it while the FR-015/FR-016 VPS runs were under way. VPS 9091 currently has only the users mcp, agent, fr011, so T3 needs the owner to create the three levels there (or it is recorded as not run).
- 2026-09-29 - rewritten and run. What changed in `tests/test_linuxctl.sh`: every target-specific value is an env var (DAEMON_URL, TOKEN, PRIV_TOKEN, UNPRIV_TOKEN, SERVICE, SVC_PATTERN, CURL_URL, LOGINS_MAY_FAIL, DOCKER_EXEC_CONTAINER, LINUXCTL) with the local-k8s values as defaults; the disk and interface are discovered from the target (`get disks --output json`, `get network interfaces --output json`) instead of `vda`/`eth0`; `LINUXCTL=/path` skips the `go build`; `set -f` keeps patterns like `ssh*` literal; a docker section runs when the privileged user can list containers. Example for a VPS stand, run on the host: `DAEMON_URL=https://localhost:9092 MCP_CA_CERT=/etc/mcpd-docker/configs/tls/mcpd.crt TOKEN=... PRIV_TOKEN=... UNPRIV_TOKEN=... LINUXCTL=$(which linuxctl) SERVICE=ssh.service SVC_PATTERN='ssh*' CURL_URL=https://example.com LOGINS_MAY_FAIL=no DOCKER_EXEC_CONTAINER=fr011-probe bash test_linuxctl.sh`. Only T3 (9091) is open.

