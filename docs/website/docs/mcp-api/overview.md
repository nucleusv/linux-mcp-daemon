---
sidebar_label: 'Overview'
sidebar_position: 1
---

# MCP API Overview

This section documents every **tool**, **resource**, and **resource template** `mcpd` exposes over the Model Context Protocol - grouped in the sidebar as **Tools**, **Resources**, and **Resource Templates**. Every page has a runnable example, foldable between the `linuxctl` form and the raw `curl` JSON-RPC call it resolves to.

## Calling the API directly with curl

`mcpd` speaks JSON-RPC 2.0 over HTTP + Server-Sent Events (SSE), not plain request/response HTTP. Every call is a two-step handshake:

1. **Open an SSE stream** (`GET /sse`, with your bearer token) and keep it open. The server immediately sends back a one-time POST endpoint:
   ```
   event: endpoint
   data: /message?session_id=8cea78ac1229e03d-179
   ```
2. **POST your JSON-RPC request** to that exact endpoint (same bearer token, `Content-Type: application/json`). The HTTP response to this POST is just an acknowledgement (`202 Accepted`) - **the actual result streams back on the SSE connection from step 1**, not in the POST's own response body.

```bash
# 1. Open the SSE stream in the background and capture the endpoint
curl -N -s --cacert mcpd.crt -H "Authorization: Bearer $MCP_TOKEN" https://localhost:9091/sse &

# 2. POST a request to the endpoint printed by step 1
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/list"}'

# 3. Watch the SSE stream from step 1 for the matching "id": "1" response
```

`mcpd.crt` is mcpd's certificate - by default the self-signed one it generated, `/etc/mcpd/configs/tls/mcpd.crt` on its host (copy it to where you run `curl`). The examples on every tool and resource page use the same form.

This is exactly what `linuxctl` does under the hood for every single command - it's a thin, schema-driven client over this same protocol, not a separate API. Each tool/resource/template page in this section shows both forms side by side.

## The two JSON-RPC methods you'll use most

- **`tools/call`** - `{"name": "<group>/<command>", "arguments": {...}}`. Used by every page under **Tools**.
- **`resources/read`** - `{"uri": "scheme://path"}`. Used by every page under **Resources** and **Resource Templates** (templates just have a filled-in `{placeholder}` in the URI).

For the friendlier, schema-discovered `linuxctl <verb> <group>` grammar shown alongside each raw call here, see the [linuxctl CLI](../linuxctl/overview) section - in particular the [Full Command Reference](../linuxctl/command-reference) for every command grouped by tool group with real captured output.

## Full Catalog

Every resource, tool, and resource template, grouped exactly as the sidebar groups them, for a quick-reference index.

### Resources, by group

**system**
- [`os://uname`](./resources/OS Uname) - Native system uname information (kernel version, node name).
- [`os://release`](./resources/OS Release) - /etc/os-release information (distribution, version).
- [`system://hostname`](./resources/System Hostname) - Native system network hostname.
- [`system://timezone`](./resources/System Timezone) - Configured IANA timezone (e.
- [`system://locale`](./resources/System Locale) - Configured locale settings (LANG, LC_*).

**network**
- [`network://interfaces`](./resources/Network Interfaces) - Network interfaces, assigned IP addresses, and detailed RX/TX traffic statistics for all interfaces.
- [`network://routes`](./resources/Network Routes) - IPv4 Routing Table (/proc/net/route).

**devices**
- [`devices://usb`](./resources/USB Devices) - Connected USB devices (lsusb equivalent).
- [`devices://pci`](./resources/PCI Devices) - Connected PCI devices (lspci equivalent).
- [`devices://dmi`](./resources/DMI Hardware Info) - Desktop Management Interface info (lshw/hwinfo equivalent).

**kernel**
- [`kernel://modules`](./resources/Kernel Modules) - Loaded kernel drivers (lsmod equivalent).

### Tools, by group

**files**
- [`files/list`](./tools/files/list) - Lists a directory like `ls -la` (permissions, owner, size, dates, symlink targets).
- [`files/read`](./tools/files/read) - Precision reading of file contents with chunking/streaming support.
- [`files/create`](./tools/files/create) - Create a new file or replace file contents.
- [`files/update`](./tools/files/update) - Programmatically edit a file by appending text or replacing specific line ranges.
- [`files/find`](./tools/files/find) - Search for files in a directory hierarchy.
- [`files/filetype`](./tools/files/filetype) - Determines a file's MIME type natively (what `file -b --mime-type` answers).
- [`files/chmod`](./tools/files/chmod) - Changes permission bits (octal or symbolic); never follows symlinks.
- [`files/chown`](./tools/files/chown) - Changes owner and/or group; never follows symlinks.

**disks**
- [`disks/free`](./tools/disks/free) - Returns disk space statistics of the filesystem holding a path, like df - in bytes, or like df -h with human_readable.
- [`disks/usage`](./tools/disks/usage) - Calculates the disk space used by a directory, like du -s - in bytes, or like du -sh with human_readable.
- [`disks/list`](./tools/disks/list) - Lists block devices and partitions.
- [`disks/mounts`](./tools/disks/mounts) - Lists mounted filesystems (device, mount point, type, options) - equivalent to `mount`/`findmnt`'s basic view.
- [`disks/performance`](./tools/disks/performance) - Retrieves granular block device I/O performance metrics (equivalent to iostat).
- [`disks/health`](./tools/disks/health) - Retrieves detailed SMART health data for a drive (equivalent to smartctl -j -a).
- [`disks/partitions`](./tools/disks/partitions) - Retrieves partition boundaries for a drive (start/size, in sectors and bytes), parsed natively from /sys/class/block - no fdisk dependency.

**processes**
- [`processes/list`](./tools/processes/list) - Lists running processes on the system.
- [`processes/top`](./tools/processes/top) - A `top -b -n 1` snapshot: load, tasks, CPU and memory header plus all of top's columns.
- [`processes/delete`](./tools/processes/delete) - Terminates a specific process by PID.

**network**
- [`network/nslookup`](./tools/network/nslookup) - Query DNS records natively.
- [`network/curl`](./tools/network/curl) - Transfer data from a URL using native HTTP client.
- [`network/arp`](./tools/network/arp) - View the system ARP cache (IP to MAC address mappings).
- [`network/ping`](./tools/network/ping) - Measure TCP reachability and latency to a host.
- [`network/connections`](./tools/network/connections) - TCP and UDP sockets with their owning processes, like `ss -tuanp`, read natively from /proc.
- [`network/trace-path`](./tools/network/trace-path) - Traces the network path to a host (equivalent to traceroute).

**memory**
- [`memory/usage`](./tools/memory/usage) - Returns memory and swap utilization information.

**system**
- [`services/manage`](./tools/services/manage) - Control systemd services (start, stop, restart, enable, disable).
- [`services/list`](./tools/services/list) - Lists systemd services with optional filtering.
- [`system/os-release`](./tools/system/os-release) - Retrieves Linux distribution and kernel version.
- [`system/packages`](./tools/system/packages) - Lists installed packages, auto-detecting the package manager (dpkg, apk; rpm-based systems aren't supported natively yet).

**logs**
- [`logs/journal-control`](./tools/logs/journal-control) - Queries the systemd journal (journalctl equivalent).
- [`logs/dmesg`](./tools/logs/dmesg) - Read the kernel ring buffer for hardware/driver logs.
- [`logs/logins`](./tools/logs/logins) - Lists login history (wraps `last`) or failed login attempts (`type: "failed"`, wraps `lastb`).

**kernel**
- [`kernel/system-control`](./tools/kernel/system-control) - Reads or writes kernel parameters (sysctl equivalent) at runtime.

**cpu**
- [`cpu/list`](./tools/cpu/list) - Retrieves CPU topology and architecture.
- [`cpu/load-average`](./tools/cpu/load-average) - Retrieves system load averages (1m, 5m, 15m).

**users**
- [`users/list`](./tools/users/list) - Lists user accounts from /etc/passwd (uid, gid, home, shell, group memberships).

**auth**
- [`auth/sudo-rules`](./tools/auth/sudo-rules) - Returns your authorized tools and privileges from mcp-sudo.

**daemon**
- [`daemon/reload-config`](./tools/daemon/reload-config) - Re-reads mcpd's config files and applies them without a restart (only for users granted it).

### Resource templates, by group

**files**
- [`file:///{path}`](./resource-templates/File Reader) - Reads any file on the system.

**devices**
- [`devices://{type}`](./resource-templates/Hardware Devices) - Hardware device metadata.

**network**
- [`network://interfaces/{name}`](./resource-templates/Network Interface Detail) - Detailed properties and RX/TX traffic statistics of a specific network interface.

**system**
- [`service://{name}/status`](./resource-templates/Service Status) - Exposes DBus service properties (ActiveState, LoadState, SubState).

**disks**
- [`disks://{name}/stats`](./resource-templates/Disk I-O Statistics) - Real-time I/O statistics for a specific block device (e.

**processes**
- [`process://{pid}/{target}`](./resource-templates/Process Introspection) - Reads process metadata from procfs.
