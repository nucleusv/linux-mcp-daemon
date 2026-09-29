# FR-027 Add Docker and crontab examples to the Medium article - after the release

- **Created:** 2026-09-29, by the owner ("lets add more examples in article ... about docker and crontabs ... but after release")
- **Related:** FR-026 (`cron/manage`, `crontabs`), FR-011/012/013 (Docker tools, released), the article https://medium.com/@nucleusv/trust-your-server-to-an-ai-agent-without-regrets-sleeping-soundly-without-ssh-f0d043ff207c, the draft in `articles/medium-docker-and-crontab-examples.md` (untracked, local)

## Description

Two new sections for the Medium article: (1) containers without handing over the Docker socket - real output and the `containers:` scoping; (2) crontabs - the design and the risks behind it. Medium blocks automated reads (HTTP 403), so the sections are written as standalone Markdown to paste in; the owner places them and matches the article's voice (send the article text if I should place them).

**Blocked on the release:** the Docker section could be published now (0.4.0 and later ship it), but the owner wants both added after the release. The crontab section is written as "what is coming" with intended output; once `crontabs` ships (FR-026) it has to be redone with **real** output captured from a stand.

## Acceptance criteria

- [ ] The release with the crontab tools is out (FR-026 in a release).
- [ ] Docker section re-checked against the released tools (commands still valid, output real, no mention of the VPN container on the test VPS or of any real host name).
- [ ] Crontab section rewritten with real captured output, "planned" wording removed, the risks section linked to the released docs page.
- [ ] The owner reviews and publishes; nothing is posted to Medium by me.
- [ ] Definition of Done (backlog/README.md) - N/A for code; the draft file stays outside git (`articles/` is untracked by design).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Every command in the text works | run each command on a stand | output matches the text | | |
| T2 | No private data | read the text for host names, IPs, the VPN container name, tokens | none present | | |

## Comments

- 2026-09-29 - created in `blocked/` (waiting on the release). Draft written from real VPS output (Docker) and from the FR-026 design (crontabs); it deliberately avoids the VPN container that shares the test VPS.
