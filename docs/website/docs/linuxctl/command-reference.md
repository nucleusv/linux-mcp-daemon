---
sidebar_label: 'Full Command Reference'
sidebar_position: 4
---

# Full Command Reference

Every tool, resource, and resource template `mcpd` exposes, with real output captured live against a running deployment (a Kubernetes-in-Docker node - `desktop-control-plane`). Mutating calls are described but mostly not executed here to keep this page's captures non-destructive - the one exception, noted inline, was a live `restart` of `kubelet.service` run once during development to verify the mutation-verb resolver, which is **not** something to run casually against a real service; everything else below actually ran read-only.

## files

```bash
$ linuxctl get files list /etc --output table
SIZE    IS_DIR   MODIFIED              NAME
0       false    2026-09-11 02:06:14   .pwd.lock
4096    true     2026-09-22 18:38:14   alternatives
...

$ linuxctl get files /etc/hosts
# Kubernetes-managed hosts file (host network).
127.0.0.1	localhost
::1	localhost ip6-localhost ip6-loopback
...

$ linuxctl create files /tmp/note.txt --content "hello"
Successfully created and wrote to /tmp/note.txt

$ linuxctl update files /tmp/note.txt --content " world" --append true
Successfully appended to /tmp/note.txt

$ linuxctl get files find /var/log --name "*.log"
/var/log/bootstrap.log
/var/log/dpkg.log

$ linuxctl get files filetype /etc/hosts
text/plain

$ linuxctl describe files /etc/hosts
=== stat ===
{"name": "hosts", "size": 273, "mode": "-rw-r--r--", "modified_time": "2026-09-23 10:23:01", "is_dir": false}
=== type ===
text/plain
```

Note: `get files list /var/log` needs the explicit `list` keyword (unlike every other bare-reachable case in this table) because `files` has two candidates that could otherwise both plausibly claim a bare path (`files/list`, a directory listing, vs `files/read`, file content), and the client can't `stat()` a path that lives on the remote system to tell which one you mean. `files/read` keeps the bare form (`get files /etc/hosts`).

## disks

```bash
$ linuxctl get disks --output table
NAME    RM      RO      SIZE_BYTES
vda     false   false   63999836160
...

$ linuxctl get disks free / --output table
FREE_BYTES    0
TOTAL_BYTES   62671097856
USE_PERCENT   99.1
USED_BYTES    62085443584

$ linuxctl get disks usage /var/log
Total size of /var/log: 332.0 KiB

$ linuxctl get disks mounts --output table
# every mounted filesystem, one row per mount

$ linuxctl get disks partitions vda --output json
[{"device": "vda1", "parent_disk": "vda", "number": 1, "start_sector": 2048, "size_sectors": 124997632, "size_bytes": 63998787584}]

$ linuxctl get disks health vda
{"smartctl": {"messages": [{"string": "/dev/vda: Unable to detect device type", "severity": "error"}]}, ...}
# a virtual disk has no SMART data to report - this is the expected real-hardware failure mode, not a tool bug

$ linuxctl get disks performance           # all devices (many results)
$ linuxctl get disks performance vda       # one device (one result) - same tool, filtered

$ linuxctl describe disks vda
[{"major": 254, "minor": 0, "device_name": "vda", ...}]
```

## processes

```bash
$ linuxctl get processes --sort_by mem --limit 3 --output table
USER   COMM             STATE   PPID   RSS_KB   CMDLINE                PID
root   kube-apiserver   S       467    336564   kube-apiserver ...     754

$ linuxctl get processes 1
[{"cmdline": "/sbin/init", "comm": "systemd", "pid": 1, "ppid": 0, "rss_kb": 18928, "state": "S", "user": "root"}]
# same tool as the bare form above, filtered to one PID - not a separate endpoint

$ linuxctl get processes top
=== load ===
Load Average: 0.70, 0.98, 0.87
=== memory ===
{"available":5941297152,"free":356151296,"total":8320299008,"used":2148139008,...}
=== top 10 processes by memory ===
[{"pid":754,"user":"root","comm":"kube-apiserver",...}, ...]

$ linuxctl describe processes 1
=== status ===
Name:	systemd
State:	S (sleeping)
...
=== cmdline ===
...
=== limits ===
...
# environ is deliberately never included - describe never leaks secret-shaped data by default

linuxctl delete processes 1234 --signal SIGTERM --privileged true   # mutating, not run here
```

## system

```bash
$ linuxctl get system hostname
desktop-control-plane

$ linuxctl get system timezone
Time zone: UTC
Abbreviation: UTC
...

$ linuxctl get system locale
Source: /etc/default/locale
LANG=C.UTF-8

$ linuxctl get system release
PRETTY_NAME="Ubuntu 24.04.5 LTS"
...

$ linuxctl get system uname
Sysname: Linux
Nodename: desktop-control-plane
...

$ linuxctl get system os-release
OS Release Info:
PRETTY_NAME="Ubuntu 24.04.5 LTS"
...

$ linuxctl get system packages --output json
[{"architecture": "arm64", "name": "apt", "version": "2.8.3"}, ...]

$ linuxctl get system services --pattern "kube*" --privileged true --output json
[{"active_state": "active", "description": "kubelet: The Kubernetes Node Agent", "load_state": "loaded", "name": "kubelet.service", "sub_state": "running"}]

$ linuxctl describe system services kubelet
{"name": "kubelet.service", "active_state": "active", "load_state": "loaded", "sub_state": "running", ...}

linuxctl restart system services nginx.service --privileged true   # mutating - see the warning above; not something to run against a real service casually
linuxctl start   system services nginx.service --privileged true
linuxctl stop    system services nginx.service --privileged true
linuxctl enable  system services nginx.service --privileged true
linuxctl disable system services nginx.service --privileged true
```

Note: `get system release`/`get system uname` are the `os://release`/`os://uname` *resources*; `get system os-release` is the separate `system/os-release` *tool* (combined kernel+distro text) - both stay reachable since neither is purely redundant with the other.

## network

```bash
$ linuxctl get network connections --output table
Netid State  Recv-Q Send-Q Local Address:Port  Peer Address:Port
tcp   LISTEN 0      4096               *:9091             *:*

$ linuxctl get network ping 8.8.8.8 --port 53
{"host": "8.8.8.8", "port": 53, "success": false, "latency_ms": 5003.5, "error": "dial tcp 8.8.8.8:53: i/o timeout"}
# egress to 8.8.8.8 is blocked in this sandboxed environment - a real network fact, not a tool bug

$ linuxctl get network curl http://127.0.0.1:9091/ping
{"status_code": 404, "status": "404 Not Found", ...}

$ linuxctl get network nslookup google.com
{"host": "google.com", "records": [...]}

$ linuxctl get network arp --output table
MASK   DEVICE   IP_ADDRESS   HW_TYPE   FLAGS   HW_ADDRESS
*      eth0     172.19.0.8   0x1       0x2     d6:ee:9a:61:0c:f8

$ linuxctl get network trace-path 1.1.1.1 --max_hops 3
traceroute to 1.1.1.1 (1.1.1.1), 3 hops max, 60 byte packets
 1  172.19.0.1 (172.19.0.1)  0.525 ms  0.025 ms  0.016 ms

$ linuxctl get network interfaces --output table
$ linuxctl get network routes --output table

$ linuxctl describe network interfaces eth0
{"addresses": ["172.19.0.7/16", ...], "mac": "b6:44:e3:c0:4f:d2", "mtu": 65535, "statistics": {...}}
```

## devices

```bash
$ linuxctl get devices usb --output table
BUS_ID   VENDOR_ID   PRODUCT_ID   MANUFACTURER                     PRODUCT
usb1     1d6b        0002         Linux 7.0.12-linuxkit vhci_hcd   USB/IP Virtual Host Controller

$ linuxctl get devices pci --output table
$ linuxctl get devices dmi
# fails with "DMI data not available on this system" on some VMs (including this one) - a real environmental limitation, not a bug
```

## kernel

```bash
$ linuxctl get kernel sysctl net.ipv4.ip_forward
net.ipv4.ip_forward = 1

$ linuxctl update kernel sysctl net.ipv4.ip_forward 1 --privileged true
net.ipv4.ip_forward = 1
# get and update resolve to the SAME tool (kernel/system-control) - it reads
# when no value is given, writes when one is, so both verbs reach it

$ linuxctl get kernel modules --output table
```

## logs

```bash
$ linuxctl get logs dmesg --privileged true
[WARNING: Output truncated to last 30KB]
...

$ linuxctl get logs journal --unit kubelet.service --lines 3 --privileged true
Sep 23 10:30:40 desktop-control-plane kubelet[471811]: ...

$ linuxctl get logs journal --lines 2 --boot true --privileged true
# current boot only (journalctl -b)

$ linuxctl get logs logins --privileged true
# on a normal host: last/lastb-equivalent login records.
# on this project's actual kind-node deployment: fails with
# "exec: \"last\": executable file not found in $PATH" - this minimal
# LinuxKit VM genuinely has no login mechanism or last/lastb binaries at
# all (see investigations/README.md) - not a tool bug.
```

## users / cpu / memory / auth

Each of these groups has exactly one tool, so no target keyword is ever needed:

```bash
$ linuxctl get users --min_uid 1000 --output table
# real OS accounts (/etc/passwd + /etc/group), UID >= 1000, never /etc/shadow

$ linuxctl get cpu --output table
# per-core topology

$ linuxctl get cpu load-average
Load Average: 1.89, 1.15, 0.95

$ linuxctl get memory usage
TYPE   TOTAL   USED   FREE   SHARED   BUFF/CACHE   AVAILABLE

$ linuxctl get auth sudo-rules
Your authorized privileged tools:
{"Tools": {...}, "Resources": {...}}
```

## Resource templates directly (bypassing the verb grammar)

The `resource <uri>` command still works as a direct escape hatch to any resource, including templates:

```bash
$ linuxctl resource file:///etc/hosts/stat
{"name": "hosts", "size": 273, "mode": "-rw-r--r--", ...}

$ linuxctl resource file:///etc/hosts/type
text/plain

$ linuxctl resource network://interfaces/eth0
{"name": "eth0", ...}

$ linuxctl resource service://kubelet.service/status
{"name": "kubelet.service", "active_state": "active", ...}
```
