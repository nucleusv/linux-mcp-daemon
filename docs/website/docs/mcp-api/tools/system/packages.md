# Packages

**Tool Name**: `system/packages`

Lists installed packages, auto-detecting the package manager (`dpkg`/Debian/Ubuntu, `apk`/Alpine) by natively parsing its database file. RPM-based systems aren't supported natively yet.

When `mcpd` runs containerized, passing `privileged: true` automatically queries the real host's installed packages instead of the daemon's own container image - see [Master Daemon Configuration](../../../configuration/daemon.md)'s `worker.containerized` setting and [Sudo Privileges](../../../configuration/mcp-sudo.md) for authorizing `privileged` access.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl get system packages
```

Output (Ubuntu 24.04 VPS, first 25 lines):
```text
756 packages installed (dpkg)
NAME                                   VERSION                                  ARCH
adduser                                3.137ubuntu1                             all
adwaita-icon-theme                     46.0-1                                   all
amd64-microcode                        3.20251202.1ubuntu0.24.04.1              amd64
apparmor                               4.0.0-beta3-0ubuntu3                     amd64
apport                                 2.28.1-0ubuntu3.8                        all
apport-core-dump-handler               2.28.1-0ubuntu3.8                        all
apport-symptoms                        0.25                                     all
appstream                              1.0.2-1build6                            amd64
apt                                    2.7.14build2                             amd64
apt-utils                              2.7.14build2                             amd64
at-spi2-common                         2.52.0-1build1                           all
at-spi2-core                           2.52.0-1build1                           amd64
atop                                   2.10.0-2ubuntu2                          amd64
base-files                             13ubuntu10                               amd64
base-passwd                            3.6.3build1                              amd64
bash                                   5.2.21-2ubuntu4                          amd64
bash-completion                        1:2.11-8                                 all
bc                                     1.07.1-3ubuntu4                          amd64
bcache-tools                           1.0.8-5build1                            amd64
bind9-dnsutils                         1:9.18.39-0ubuntu0.24.04.7               amd64
bind9-host                             1:9.18.39-0ubuntu0.24.04.7               amd64
bind9-libs                             1:9.18.39-0ubuntu0.24.04.7               amd64
bolt                                   0.9.7-1                                  amd64
...
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
# 1. Open the SSE stream (in the background) and capture the one-time POST endpoint
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

# 2. POST the tools/call request to that endpoint
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "system/packages", "arguments": {}}}'

# 3. The result arrives on the SSE stream opened in step 1
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "756 packages installed (dpkg)\nNAME                                   VERSION                                  ARCH\nadduser                                3.137ubuntu1                             all\nadwaita-icon-theme                     46.0-1                                   all\namd64-microcode                        3.20251202.1ubuntu0.24.04.1              amd64\napparmor                               4.0.0-beta3-0ubuntu3                     amd64\napport                                 2.28.1-0ubuntu3.8                        all\napport-core-dump-handler               2.28.1-0ubuntu3.8                        all\napport-symptoms                        0.25                                     all\nappstream                              1.0.2-1build6                            amd64\napt                                    2.7.14build2                             amd64\napt-utils                              2.7.14build2                             amd64\nat-spi2-common                         2.52.0-1build1                           all\nat-spi2-core                           2.52.0-1build1                           amd64\natop                                   2.10.0-2ubuntu2                          amd64\nbase-files                             13ubuntu10                               amd64\nbase-passwd                            3.6.3build1                              amd64\nbash                                   5.2.21-2ubuntu4                          amd64\nbash-completion                        1:2.11-8                                 all\nbc                                     1.07.1-3ubuntu4                          amd64\nbcache-tools                           1.0.8-5build1                            amd64\nbind9-dnsutils                         1:9.18.39-0ubuntu0.24.04.7               amd64\nbind9-host                             1:9.18.39-0ubuntu0.24.04.7               amd64\nbind9-libs                             1:9.18.39-0ubuntu0.24.04.7               amd64\nbolt                                   0.9.7-1                                  amd64\n..."
      }
    ]
  }
}
```

</details>

