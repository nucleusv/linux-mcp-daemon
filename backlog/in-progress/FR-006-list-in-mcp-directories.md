# FR-006 List mcpd in MCP directories

- **Created:** 2026-09-26, by the owner ("давай зарегистрируемся")
- **Related:** `investigations/feature-backlog.md` #26 (listing notes), `server.json`, `glama.json`

## Description

Make mcpd findable where people look for MCP servers: the official MCP Registry (other catalogs such as PulseMCP and Smithery pull from it), Glama, mcp.so, PulseMCP and the awesome-mcp-servers list.

mcpd is self-hosted and network-only, so for the official registry it is described as a **remote** (`remotes`, SSE at `https://{host}:9091/sse`, the host and an `Authorization` header supplied by the user) - no package verification is needed, only the `io.github.nucleusv` namespace, proven by logging into GitHub with `mcp-publisher`. An OCI package entry (`ghcr.io/nucleusv/linux-mcp-daemon`) would need the image labeled `io.modelcontextprotocol.server.name` and a new release - left for later.

Account sign-ups, logins, submissions and the pull request are done by the owner or with the owner's explicit OK each time: they publish in the owner's name.

Pitch used everywhere: "Self-hosted Linux admin over MCP: remote over HTTPS, a separate worker per call running as the caller's OS user, root granted per tool within path/network limits, audit log." Categories: System Administration, Monitoring, Security. 38 tools, Apache-2.0, docs link.

## Acceptance criteria

- [x] `server.json` (remote, SSE, `host` variable, secret `Authorization` header) and `glama.json` (maintainer `nucleusv`) in the repo root, valid JSON.
- [x] Both files committed and pushed (0b6ff7a).
- [x] Published to the official MCP Registry with `mcp-publisher` (GitHub login by the owner); visible at registry.modelcontextprotocol.io.
- [x] Listed on Glama (owner's account, repo claimed).
- [ ] Submitted to mcp.so and PulseMCP.
- [ ] Pull request to punkpeye/awesome-mcp-servers opened (owner's OK).
- [ ] Each listing's URL recorded below.

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Files valid | `python3 -c json.load` on both; `mcp-publisher validate` | valid | 2026-09-26, local | ✅ json ok; `mcp-publisher 1.8.1 validate`: "server.json is valid" |
| T2 | Registry | `curl https://registry.modelcontextprotocol.io/v0/servers?search=io.github.nucleusv/linux-mcp-daemon` | our entry, version 0.3.4 | 2026-09-26 | ✅ `io.github.nucleusv/linux-mcp-daemon 0.3.4`, status active, remote `https://{host}:9091/sse` |
| T3 | Glama | glama.ai listing + release | listing present, maintainer nucleusv | 2026-09-27 | ✅ claimed, build test passed (mcpd stdio), release 0.3.5, TDQS A |
| T4 | mcp.so / PulseMCP | site search | listing present | | |
| T5 | awesome list | PR link | open or merged | | |

## Comments

- 2026-09-26 - created, in progress. Opened the catalog pages in the owner's browser. Glama's "Add Server" asks for an account (sign-up with captcha) - owner signs up, preferably with GitHub. Glama search for `linux-mcp-daemon` finds nothing yet.
- 2026-09-26 - committed and pushed server.json + glama.json (0b6ff7a, VPS address removed from FR-003 before commit). Installed mcp-publisher 1.8.1 (brew); `mcp-publisher validate` against registry.modelcontextprotocol.io: valid. Next: owner runs `mcp-publisher login github`.
- 2026-09-26 - Glama: owner signed in and submitted "Runs from source" (name "Linux MCP daemon", repo github.com/nucleusv/linux-mcp-daemon, description as in the pitch). Pending review; Glama then emails instructions for a Dockerfile used by its automated safety/quality checks - only servers passing them are indexed for search.
- 2026-09-26 - official MCP Registry: owner authorized the GitHub device login (`mcp-publisher login github`), then `mcp-publisher publish`: "Successfully published io.github.nucleusv/linux-mcp-daemon version 0.3.4" at 19:53:47Z; the registry API returns it as active. Listing: https://registry.modelcontextprotocol.io/v0/servers?search=io.github.nucleusv/linux-mcp-daemon
- 2026-09-26 - owner asked to keep the registry entry current: added "Releasing vX.Y.Z" to CLAUDE.md - bump `server.json` version with the others, then `mcp-publisher publish` after the tag and verify via the registry API.
- 2026-09-26 - Glama: owner claimed the server; added card (end of README) and score (top badge row) badges to README. Dockerfile config with mcpd over HTTP + `mcp-remote` worked locally (37 tools via mcp-proxy) but Glama rejects `mcp-remote` ("must build and run the server locally, not proxy"). Not using another bridge to get around that rule; native stdio mode is FR-007, Glama's build waits for it. Registry description changed in server.json (not yet republished): "AI agent access to Linux servers without SSH: scoped tools, per-call isolation, root only if granted".
- 2026-09-27 - registry at 0.3.5 with the new description; Glama release 0.3.5 and README images fixed for Glama. Left: mcp.so, PulseMCP (may pick up from the registry), awesome-mcp-servers PR (needs the owner's OK).
