# FR-010 Richer descriptions for the 8 lowest-rated tools (Glama TDQS)

- **Created:** 2026-09-27, by the owner (Glama schema page: "посмотри жёлтые и красные и как можно улучшить")
- **Related:** `internal/rpc/tools.go` (HandleToolsList), the tools' docs pages, FR-006 (Glama)

## Description

Glama's Tool Definition Quality Score for v0.3.5 is 3.9/5 on average over 37 tools. Two tools are C (red), six are B (yellow), the rest A:

| Tool | Score | Description today | Length |
|---|---|---|---|
| network/nslookup | C 2.6 | "Query DNS records natively." | 27 |
| files/find | C 2.7 | "Search for files in a directory hierarchy." | 42 |
| files/create | B 3.1 | "Create a new file or replace file contents." | 43 |
| network/arp | B 3.2 | "View the system ARP cache (IP to MAC address mappings)." | 55 |
| processes/delete | B 3.2 | "Terminates a specific process by PID." | 37 |
| network/ping | B 3.3 | "Measure TCP reachability and latency to a host." | 47 |
| services/list | B 3.3 | "Lists systemd services with optional filtering. Output includes ActiveState, LoadState, and SubState." | 101 |
| files/read | B 3.4 | "Precision reading of file contents with chunking/streaming support." | 67 |

The best-rated tools (processes/top 4.7, disks/list 4.7, network/connections 4.5) have 450-550-character descriptions that say what comes back and in what shape, when to use the tool and which neighbouring tool to use instead, and what it does or won't do. The low ones are one line. Several parameters have no description at all: `network/nslookup.host`, `network/arp.interface`, `network/ping.host`, `network/ping.timeout`.

Glama rates the definitions of a release, so the new scores appear with the next release.

### Proposed descriptions (checked against the code)

- **network/nslookup** - "Resolves a hostname with this host's system resolver, natively (no dig/nslookup binary). By default returns every record type it can find (ANY); record_type limits it to one of A, AAAA, CNAME, TXT, MX or NS. Each record comes back with its type and value (json/yaml with output_format); a name that doesn't exist is an error, not an empty list. Use it to see what a dependency resolves to from this server; whether a port answers is network/ping, the route is network/trace-path." Params: `host` "Hostname to resolve, e.g. api.example.com".
- **files/find** - "Searches a directory tree like find(1), without following symlinks. Filters: name (a glob such as '*.log'), type (f file, d directory, l symlink), size ('+100M' larger than 100 MiB, '-1k' smaller than 1 KiB) and mtime ('+7' older than 7 days); max_depth limits recursion. Returns each match's path, size and type. /proc, /sys, /dev and /run are skipped; it starts at / unless path is set, so give path and max_depth to keep it fast. Use it to find large or old files; for a directory's total size use disks/usage, for one directory's listing files/list. With privileged: true it searches as root, only inside the paths granted in mcp-sudo.yaml."
- **files/create** - "Writes a file: creates it with the given content, or replaces an existing file's whole content. Missing parent directories are created (mode 0755), a new file gets 0644. Without content it works like touch - an empty file, or just a new modification time for an existing one. To change some lines use files/update, to append files/update with append: true. With privileged: true it writes as root, only inside the paths granted in mcp-sudo.yaml, and a symlink anywhere on the path is refused. Changes the host; audit-logged."
- **network/arp** - "Lists this host's ARP cache from /proc/net/arp: each neighbour's IP address, MAC address, interface and flags. interface filters by device (e.g. eth0). Use it to see which machines this host has recently talked to on the local network, or to check a duplicate IP / wrong MAC; for addresses and state of this host's own interfaces use the network://interfaces resource."
- **processes/delete** - "Sends a signal to one process by PID - SIGTERM by default (a polite stop the process can handle), or SIGKILL, SIGHUP and other named signals via signal. Refuses PID 1 and mcpd itself. Without privileged it can only signal the caller's own processes; privileged: true (granted in mcp-sudo.yaml) signals any. Find the PID first with processes/list or processes/top. Changes the host; audit-logged."
- **network/ping** - "Checks whether a TCP port on a host accepts connections and how long the connection takes - a TCP ping, not ICMP, so it works where ICMP is blocked and tests the actual service port. port defaults to 80, timeout (seconds) to 5. Returns reachable yes/no and the latency. Use it to check a dependency's port from this server; for name resolution use network/nslookup, for the network path network/trace-path, for this host's own open sockets network/connections." Params: `host` "Hostname or IP address", `timeout` "Seconds to wait for the connection (default 5)".
- **services/list** - "Lists systemd units over D-Bus with their LoadState, ActiveState and SubState (like systemctl list-units). Filter by name with a wildcard pattern ('*ssh*', 'nginx*') and by state - active_state 'failed' is the quick way to find what is broken. Output as text, json, table or wide. Use it to find a unit's exact name and state; for one unit's details read the service://{name}/status resource, for its logs logs/journal-control, to start/stop/restart it services/manage."
- **files/read** - "Reads a text file: by default the first 10 KiB; start_line/end_line read a range of lines (1-indexed, inclusive), offset/limit a range of bytes - use them to page through large logs. Binary files are refused with their detected type. Use files/list to see what is in a directory and files/find to locate a file first. With privileged: true it reads as root, only inside the paths granted in mcp-sudo.yaml, never through a symlink."

## Acceptance criteria

- [ ] The 8 descriptions and the 4 missing parameter descriptions updated in `internal/rpc/tools.go`, every statement checked against the tool's code.
- [ ] Their docs pages (`docs/website/docs/mcp-api/tools/...`) show the new descriptions.
- [ ] `tools/list` over MCP returns the new text (live check).
- [ ] Released; Glama TDQS for the release shows all 8 at A (≥ 3.6) - or the remaining gap is recorded.
- [ ] Definition of Done (backlog/README.md).

## Tests

| # | Checks | How | Expected | Run | Result |
|---|---|---|---|---|---|
| T1 | Registry builds and lists | `go test ./internal/rpc/...`; `linuxctl get mcp-api tools` | new descriptions | | |
| T2 | No false claims | each description re-read against the tool's code | every statement true | | |
| T3 | Docs | `bash scripts/check_docs.sh`; pages updated | passes | | |
| T4 | Glama | TDQS page for the next release | 8 tools A | | |

## Comments

- 2026-09-27 - created from Glama's TDQS page (v0.3.5: 3.9/5 average; nslookup 2.6, find 2.7, create 3.1, arp 3.2, delete 3.2, ping 3.3, services/list 3.3, read 3.4). Glama shows scores only, no per-dimension reasons; the pattern (length/usage guidance/alternatives/side effects) is inferred from the A-rated tools. Discovery score went 33% → 83% after the Glama release.
