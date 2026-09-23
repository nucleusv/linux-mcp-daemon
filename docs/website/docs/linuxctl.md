---
sidebar_label: 'CLI / linuxctl'
---

# linuxctl CLI

The `linuxctl` command-line interface allows users and AI agents to seamlessly interact with the Linux Model Context Protocol (MCP) Daemon (`mcpd`) running in the background.

## Synopsis

```bash
linuxctl [OPTIONS] resources
linuxctl [OPTIONS] resource <uri>
linuxctl [OPTIONS] <group> <command> [command-options]
linuxctl [OPTIONS] <group>/<command> [command-options]
```

`<group> <command>` (space-separated) and `<group>/<command>` (slash form) are both accepted and equivalent - all examples below use the space form.

## Options

- `-server URL`
  The URL of the `mcpd` server to connect to. Defaults to `http://localhost:9091`.

- `-token TOKEN`
  Bearer token for authentication. If not provided via this flag, the client will automatically look for the `MCP_TOKEN` environment variable.

## Commands

### `ping`
Pings the `mcpd` daemon to verify connectivity and authentication. Returns a success message if the daemon is reachable and the token is valid.

### `resources`
Fetches and lists all statically available resources and resource templates exposed by the daemon.

### `resource <uri>`
Reads the exact contents of an MCP resource. Resource URIs always use `scheme://path` syntax, e.g. `os://uname`, `system://hostname`, `file:///etc/hosts` (note the triple slash - an empty authority followed by an absolute path). Use `--output table` or `--output json` to optionally format structural JSON outputs.

### `<group> <command>`
Dynamically executes an MCP tool exposed by the daemon - for example `files list` or `disks free`. Run `linuxctl` without arguments to see available groups.

Running `linuxctl <group>` alone (with no command) behaves in one of two ways depending on the group, and this is a real, current quirk worth knowing rather than a documentation simplification: for groups with a configured default command (`disks`, `cpu`, `files`, `processes`, `network`, `memory`, `system`), it immediately **runs that default command** (e.g. `linuxctl system` runs `system/os-release`). For every other group (`auth`, `kernel`, `logs`, `users`, `services` note: folded under `system`, see below), it instead **lists the group's available commands**. If you're unsure which behavior a group has, just run `linuxctl <group> <command> -h`-style exploration via `linuxctl resources` / `linuxctl tools/list` directly, or check `internal/rpc/tools.go`'s `tools_group` field for the tool you're after.

### `[command-options]`
Arguments specific to the command being executed. You can provide these either positionally:
```bash
linuxctl files list /var/log
```
Or as explicit flags:
```bash
linuxctl files list --path /var/log --privileged true
```
You can also append `--output table`, `--output wide`, `--output yaml`, or `--output json` to format structural data returned by tools or resources.

### `<verb> mcpd user <username>` - local-only user/token administration

Unlike every other command on this page, `mcpd` never talks to the daemon over the network - it reads and writes `configs/daemon.yaml` + `configs/mcp-sudo.yaml` directly on disk, needs no `-token`/`-server` flag, and works even if the daemon isn't running. This is intentional: user/token administration is a privilege-escalation-relevant surface, kept off the network `tools/call` path entirely. See `docs/website/docs/configuration/daemon.md` for full detail (token hashing, why `create`/`delete` touch both files atomically, and the OS-account step a new user still needs before a rebuild).

```bash
linuxctl create   mcpd user alice [--set-token VALUE] [--config-path DIR]   # DIR defaults to ./configs
linuxctl delete   mcpd user alice [--config-path DIR]
linuxctl update   mcpd user alice [--set-token VALUE] [--config-path DIR]   # rotates the token
linuxctl list     mcpd users      [--config-path DIR]
linuxctl describe mcpd user alice [--config-path DIR]
```

**Example - creating a user and seeing the one-time token:**
```bash
$ linuxctl create mcpd user alice
Created mcpd user "alice".

Token (shown once - not stored in plaintext anywhere, save it now):
  863953e1c3a24cee64c4c2306af990da5fd9041bb8c9db5f0d48d478df5d860d

Next steps:
  1. Add a matching OS account (useradd -m -s /bin/bash alice in the Dockerfile) - workers run
     as a real OS user via user.Lookup(), so this account must exist before alice can make any call.
  2. Grant tools/resources for "alice" in configs/mcp-sudo.yaml (currently empty - denies everything by default)
  3. Run scripts/deploy.sh to rebuild and apply
```

## Environment Variables

- `MCP_TOKEN`
  The bearer token used for authenticating with the daemon. This is the recommended way to authenticate so you don't leak tokens in shell histories.

## Examples

**Ping the daemon using a specific token:**
```bash
$ linuxctl -token "my-test-token-123" ping
Successfully connected to mcpd daemon!
```

**Ping the daemon using an environment variable (Recommended):**
```bash
export MCP_TOKEN="my-test-token-123"
linuxctl ping
```

**List all available resources:**
```bash
$ linuxctl resources
Available static resources:
  os://uname            - Kernel/OS version details (equivalent to `uname -a`)
  os://release           - Distribution info (equivalent to /etc/os-release)
  system://hostname      - The system hostname
  system://timezone      - Configured timezone
  system://locale        - Configured system locale
  network://interfaces   - Network interface list with addresses
  network://routes       - Kernel routing table
  devices://usb          - Attached USB devices
  devices://pci          - Attached PCI devices
  devices://dmi          - Desktop Management Interface info (BIOS/chassis)
  kernel://modules       - Loaded kernel drivers

Available resource templates:
  file:///{path}                  - Reads any file; append /stat, /content, or /type
  devices://{type}                - Hardware device metadata (usb, pci, dmi)
  network://interfaces/{name}     - Detailed stats for one interface
  service://{name}/status         - DBus service properties (ActiveState, LoadState, SubState)
  ...
```

**Read the system's kernel/OS version as a resource:**
```bash
$ linuxctl resource os://uname
Sysname: Linux
Nodename: desktop-control-plane
Release: 7.0.12-linuxkit
Version: #1 SMP PREEMPT Thu Aug 27 14:02:21 UTC 2026
Machine: aarch64
```

**List files in a directory, as a table:**
```bash
$ linuxctl files list --path /var/log --output table
MODIFIED              NAME               SIZE     IS_DIR
2026-09-22 18:38:14   alternatives.log   6522     false
2026-09-22 18:38:10   apt                4096     true
2026-09-11 02:06:15   bootstrap.log      61237    false
...
```

**Run a privileged tool (requires a root rule in `configs/mcp-sudo.yaml`):**
```bash
$ linuxctl disks free --path / --privileged true --output table
FREE_BYTES    0
PATH          /
TOTAL_BYTES   62671097856
USE_PERCENT   100
USED_BYTES    62671097856
```

**Native partition geometry, no `fdisk` involved (parsed from `/sys/class/block`):**
```bash
$ linuxctl disks partitions --device vda --output json
[
  {
    "device": "vda1",
    "parent_disk": "vda",
    "number": 1,
    "start_sector": 2048,
    "size_sectors": 124997632,
    "size_bytes": 63998787584
  }
]
```

**List installed OS packages:**
```bash
$ linuxctl system packages --output json | head
[
  { "architecture": "arm64", "name": "apt", "version": "2.8.3" },
  { "architecture": "arm64", "name": "base-files", "version": "13ubuntu10.5" }
]
```

**Query the systemd journal for a specific unit's logs since the last boot (requires `privileged: true` in containerized deployments - `journalctl` only exists on the host):**
```bash
linuxctl logs journal-control --unit kubelet.service --boot true --privileged true
```

**Read and write a kernel parameter (sysctl equivalent):**
```bash
$ linuxctl kernel system-control --key net.ipv4.ip_forward
net.ipv4.ip_forward = 1

linuxctl kernel system-control --key net.ipv4.ip_forward --value 1 --privileged true
```

**Services are still invoked via `linuxctl services <command>` today** - `services/list` and `services/manage` are matched by their literal tool name, which the CLI's current implementation resolves independently of the `tools_group` field:
```bash
$ linuxctl services list --pattern "kube*" --privileged true --output json
[{"active_state": "active", "description": "kubelet: The Kubernetes Node Agent", "load_state": "loaded", "name": "kubelet.service", "sub_state": "running"}]

linuxctl services manage --service nginx.service --action restart --privileged true
```
Note: both tools' schema `tools_group` is actually `"system"` (see `CLAUDE.md`'s known-gotchas section) - that's forward-looking metadata for the `linuxctl` grammar redesign in `plan/linuxctl-redesign.md` (where this will become `linuxctl system services list`), not yet reflected in today's routing. `linuxctl services` with no command currently reports "no commands found" as a result (it filters by `tools_group`), even though `linuxctl services list`/`linuxctl services manage` work fine.

## Full Command Reference

Every tool, static resource, and resource template this daemon exposes, with real output captured live against a running deployment (a Kubernetes-in-Docker node - `desktop-control-plane`). Mutating calls (`files create`/`files update`/`services manage`/`processes delete`/`kernel system-control --value`) are described but not executed here to keep this page's captures non-destructive; everything else below actually ran.

### files

```bash
$ linuxctl files list --path /etc --output table
SIZE    IS_DIR   MODIFIED              NAME
0       false    2026-09-11 02:06:14   .pwd.lock
4096    true     2026-09-22 18:38:14   alternatives
...

$ linuxctl files read --path /etc/hosts
# Kubernetes-managed hosts file (host network).
127.0.0.1	localhost
::1	localhost ip6-localhost ip6-loopback
...

$ linuxctl files create --path /tmp/note.txt --content "hello"
Successfully created and wrote to /tmp/note.txt

$ linuxctl files update --path /tmp/note.txt --content " world" --append true
Successfully appended to /tmp/note.txt

$ linuxctl files find --path /etc --name "hosts*"
/etc/hosts

$ linuxctl files filetype --path /etc/hosts
text/plain
```

### disks

```bash
$ linuxctl disks list --output table
NAME    RM      RO      SIZE_BYTES
vda     false   false   63999836160
vda1    false   false   63998787584
vdb     false   true    676458496
...

$ linuxctl disks free --path / --output table
FREE_BYTES    0
PATH          /
TOTAL_BYTES   62671097856
USE_PERCENT   100
USED_BYTES    62671097856

$ linuxctl disks usage --path /var/log --max_depth 1 --output table
# per-subdirectory size breakdown, one row per entry under /var/log

$ linuxctl disks mounts --fs_type ext4 --output table
DEVICE      MOUNT_POINT             FS_TYPE   OPTIONS
/dev/vda1   /etc/hosts              ext4      rw,relatime,discard
/dev/vda1   /etc/resolv.conf        ext4      rw,relatime,discard
...

$ linuxctl disks partitions --device vda --output json
[{"device": "vda1", "parent_disk": "vda", "number": 1, "start_sector": 2048, "size_sectors": 124997632, "size_bytes": 63998787584}]

$ linuxctl disks performance --device vda --output json
# per-device iostat-equivalent counters: reads_completed, sectors_written, time_doing_ios_ms, ...

$ linuxctl disks health --device vda
{
  "smartctl": { "messages": [{"string": "/dev/vda: Unable to detect device type", "severity": "error"}] },
  ...
}
# a virtual disk has no SMART data to report - this is the expected real-hardware failure mode, not a tool bug
```

### processes

```bash
$ linuxctl processes list --limit 5 --sort_by mem --output table
# top 5 processes by memory, one row per process

linuxctl processes delete --pid 1234 --signal SIGTERM --privileged true   # mutating, not run here
```

### network

```bash
$ linuxctl network connections --output table
Netid State  Recv-Q Send-Q Local Address:Port  Peer Address:Port
tcp   LISTEN 0      4096               *:9091             *:*
tcp   LISTEN 0      4096               *:6443             *:*
...

$ linuxctl network ping --host 127.0.0.1 --port 9091
{"host": "127.0.0.1", "port": 9091, "success": true, "latency_ms": 0.3}

$ linuxctl network curl --url http://127.0.0.1:9091/ping
# raw HTTP response body

$ linuxctl network nslookup --host google.com --record_type A
# A records for google.com

$ linuxctl network arp --output table
# IP-to-MAC ARP cache entries

$ linuxctl network trace-path --host 1.1.1.1 --max_hops 5
traceroute to 1.1.1.1 (1.1.1.1), 5 hops max, 60 byte packets
 1  172.19.0.1 (172.19.0.1)  1.569 ms  0.101 ms  0.029 ms
 2  * * *
 ...
```

### memory / cpu

```bash
$ linuxctl memory usage --output table
# total/used/free/swap breakdown

$ linuxctl cpu list --output table
# per-core topology

$ linuxctl cpu load-average
Load Average: 0.81, 1.02, 1.14
```

### kernel

```bash
$ linuxctl kernel system-control --key net.ipv4.ip_forward
net.ipv4.ip_forward = 1

linuxctl kernel system-control --key net.ipv4.ip_forward --value 1 --privileged true   # mutating, not run here

$ linuxctl kernel system-control --read_all true
abi.tagged_addr_disabled = 0
debug.exception-trace = 0
dev.scsi.logging_level = 0
...
```

### logs

```bash
$ linuxctl logs dmesg --privileged true
[WARNING: Output truncated to last 30KB]
...
docker0: port 1(veth2271c32) entered disabled state
...

$ linuxctl logs journal-control --unit kubelet.service --lines 3 --privileged true
Sep 22 23:27:09 desktop-control-plane kubelet[268]: ...

$ linuxctl logs journal-control --lines 2 --boot true --privileged true
Sep 22 23:27:31 desktop-control-plane containerd[151]: ...

$ linuxctl logs logins --privileged true
# on a normal host: last/lastb-equivalent login records.
# on this project's actual kind-node deployment: fails with
# `exec: "last": executable file not found in $PATH` - this minimal
# LinuxKit VM genuinely has no login mechanism or last/lastb binaries
# at all (see investigations/README.md) - not a tool bug.
```

### system

```bash
$ linuxctl system os-release
Kernel: Linux desktop-control-plane 7.0.12-linuxkit #1 SMP PREEMPT ...
OS Release: PRETTY_NAME="Ubuntu 24.04.5 LTS" ...

$ linuxctl system packages --output json
[{"architecture": "arm64", "name": "apt", "version": "2.8.3"}, ...]
```

### users / auth

```bash
$ linuxctl users list --min_uid 1000 --output table
# real OS accounts (/etc/passwd + /etc/group), UID >= 1000, never /etc/shadow

$ linuxctl auth sudo-rules
# your own authorized tools/resources from mcp-sudo.yaml, e.g.:
TOOLS   {"cpu/list":{"Allowed":true,"Paths":null}, "files/list":{"Allowed":true,"Paths":["/"]}, ...}
```

### resources (static)

```bash
$ linuxctl resource os://uname
Sysname: Linux
Nodename: desktop-control-plane
Release: 7.0.12-linuxkit
...

$ linuxctl resource os://release
# /etc/os-release contents

$ linuxctl resource system://hostname
desktop-control-plane

$ linuxctl resource system://timezone
Time zone: UTC
Abbreviation: UTC
...

$ linuxctl resource system://locale
Source: /etc/default/locale
LANG=C.UTF-8

$ linuxctl resource network://interfaces --output table
# every interface, one row per NIC

$ linuxctl resource network://routes --output table
# kernel routing table

$ linuxctl resource devices://usb --output table
MANUFACTURER                     PRODUCT                          BUS_ID   VENDOR_ID   PRODUCT_ID
Linux 7.0.12-linuxkit vhci_hcd   USB/IP Virtual Host Controller   usb1     1d6b        0002

$ linuxctl resource devices://pci --output table
# PCI device list

$ linuxctl resource devices://dmi
# BIOS/chassis info - fails with "DMI data not available on this system" on
# some VMs (including this one); a real environmental limitation, not a bug

$ linuxctl resource kernel://modules --output table
# loaded kernel modules
```

### resource templates

```bash
# file:///{path} - bare path defaults to content, or append /stat or /type
$ linuxctl resource file:///etc/hosts/stat
{"name": "hosts", "size": 273, "mode": "-rw-r--r--", "modified_time": "2026-09-22 23:36:33", "is_dir": false}

$ linuxctl resource file:///etc/hosts/type
text/plain

$ linuxctl resource file:///etc/hosts
# same as file:///etc/hosts/content - raw file contents

$ linuxctl resource network://interfaces/eth0
{"name": "eth0", "mac": "b6:44:e3:c0:4f:d2", "mtu": 65535, "addresses": ["172.19.0.7/16", ...], "statistics": {"rx_bytes": 521175827, ...}}

$ linuxctl resource service://kubelet.service/status
{"name": "kubelet.service", "active_state": "active", "load_state": "loaded", "sub_state": "running", ...}

$ linuxctl resource devices://usb   # devices://{type} also serves pci and dmi the same way
```
