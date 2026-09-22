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
