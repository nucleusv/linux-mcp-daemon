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
- [ ] Both files committed and pushed.
- [ ] Published to the official MCP Registry with `mcp-publisher` (GitHub login by the owner); visible at registry.modelcontextprotocol.io.
- [ ] Listed on Glama (owner's account, repo claimed).
- [ ] Submitted to mcp.so and PulseMCP.
- [ ] Pull request to punkpeye/awesome-mcp-servers opened (owner's OK).
- [ ] Each listing's URL recorded below.

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Files valid | `python3 -c json.load` on both; `mcp-publisher validate` if available | valid | 2026-09-26, local | ✅ json ok (description 94 chars) |
| T2 | Registry | `curl https://registry.modelcontextprotocol.io/v0/servers?search=linux-mcp-daemon` | our entry, version 0.3.4 | | |
| T3 | Glama | glama.ai search `linux-mcp-daemon` | listing present, maintainer nucleusv | 2026-09-26 | ❌ not indexed yet (baseline) |
| T4 | mcp.so / PulseMCP | site search | listing present | | |
| T5 | awesome list | PR link | open or merged | | |

## Comments

- 2026-09-26 - created, in progress. Opened the catalog pages in the owner's browser. Glama's "Add Server" asks for an account (sign-up with captcha) - owner signs up, preferably with GitHub. Glama search for `linux-mcp-daemon` finds nothing yet.
