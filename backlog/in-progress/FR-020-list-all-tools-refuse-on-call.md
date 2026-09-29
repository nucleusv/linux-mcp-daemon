# FR-020 Every tool is listed to every user; an ungranted or misused call is an error, not an absence

- **Created:** 2026-09-29, by the owner ("all tools must be present, they must give errors if they are wrongly used" - after reading the 0.4.0 note that ungranted docker tools "stay out of tools/list")
- **Related:** `internal/rpc/docker.go` (`dockerTools`), `internal/rpc/tools.go` (`HandleToolsList`, the `daemon/reload-config` block), `internal/rpc/docker_test.go` (`TestDockerToolsListingFollowsGrants`), `docs/website/docs/configuration/mcp-sudo.md`, FR-019 (0.4.0 release notes)

## Description

Today the docker tools (`docker/*`) and `daemon/reload-config` are added to `tools/list` only for users whose grant allows them: the reasoning was "they couldn't call it anyway". The owner's rule is the opposite: **the tool surface is the same for everyone**, and a call the user may not make fails with a clear error naming the missing grant. Hiding a tool makes a wrong call look like a typo (`no group "docker"`), and it makes the docs' example output depend on who is asking. The call-side refusals already exist (`prepareDockerCall`: `user X is not authorized to run docker/exec: ... needs allowed: true ...`; `daemon/reload-config`: `... (grant it in mcp-sudo.yaml)`), so this is the listing side only, plus the tests and docs that state the old behaviour. The docker resource templates were already listed to everyone (only reads are refused) - this makes the tools match them.

## Acceptance criteria

- [ ] `tools/list` returns every `docker/*` tool and `daemon/reload-config` to every authenticated user, granted or not.
- [ ] Calling one without its grant returns the existing "not authorized ... needs a grant in mcp-sudo.yaml" error (no socket dial, no worker spawn), for every docker tool and for reload-config.
- [ ] Descriptions no longer say "being able to call it at all means it was granted"; they say a call is refused without the grant.
- [ ] `linuxctl` with an ungranted user: `get docker containers` prints the not-authorized error (not `no group "docker"` / `no read target`), and tab completion lists the docker group.
- [ ] `TestDockerToolsListingFollowsGrants` is replaced by tests that assert the new behaviour; docs (`mcp-sudo.md`, the docker tool pages, the 0.4.x release notes) say the same.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Listing | Go unit test: `dockerTools` and the `HandleToolsList` path for a user with no docker grants | all docker tools and reload-config present | | |
| T2 | Call refused | Go unit test: `prepareDockerCall` for an ungranted user, each docker tool; reload-config call | not-authorized error naming the grant | | |
| T3 | linuxctl, ungranted | live, VPS 9092 as testuser: `get docker containers`, `exec docker fr011-probe id` | not-authorized error naming mcp-sudo.yaml; completion lists docker | | |
| T4 | No regression, granted | live, VPS 9092 as privileged and 9091 as fr011 | same output as before | | |
| T5 | Docs/build | `check_docs.sh`, `go test ./...`, `GOOS=linux go build ./...` | pass | | |

## Comments

- 2026-09-29 - created. Found live: as `testuser` (no docker grants) `get mcp-api tools` lists no `docker/*` tools while `get mcp-api resources` lists all four docker templates; `get docker containers` answered `no read target "containers" in group "docker"`. The owner's decision above reverses the documented "not even listed" behaviour, so the 0.4.0 release notes sentence (already corrected once to describe today's behaviour) becomes wrong again after this ships.
