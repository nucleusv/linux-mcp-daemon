# kernel/system-control

Reads and writes kernel parameters at runtime - the `sysctl` equivalent - natively through `/proc/sys`, the same files the `sysctl` binary uses. No external binary is run, so there is no command line to inject options into.

## Behavior
- **Read:** `key` in dotted (`net.ipv4.ip_forward`) or slash form (`net/ipv4/conf/eth0.100/rp_filter` - needed when a component itself contains a dot, as with VLAN interfaces). Output matches `sysctl`: `net.ipv4.ip_forward = 1`.
- **Read a subtree:** a key naming a directory (`net.ipv4`) prints every parameter under it, sorted.
- **Read all:** `read_all: true` walks all of `/proc/sys`, like `sysctl -a`. Unreadable entries (write-only triggers such as `vm.compact_memory`, root-only ones) are skipped, as `sysctl -a` does.
- **Write:** `value` writes the parameter, then reports it as the kernel now holds it (like `sysctl -w`). Requires `privileged: true`. A key that doesn't exist fails - nothing is ever created.
- Keys are validated and resolved strictly inside `/proc/sys`; `..` and leading `-` are rejected.

## Restricting writes
Writing some parameters is equivalent to running code as root (`kernel.core_pattern`, `kernel.modprobe`, ...). An optional `sysctl:` block on this tool's entry in `mcp-sudo.yaml` restricts writes per user - checked in the daemon before any worker runs. Absent = unrestricted.

```yaml
kernel/system-control:
  allowed: true
  sysctl:
    read_only: true                              # refuse every write
    # or:
    write_keys: ["net.ipv4.ip_forward", "vm.*"]  # only these keys ("*" = one dotted component)
```

## Parameters
- `key` (string): parameter name or subtree. Required unless `read_all`.
- `value` (string, optional): value to write; single line.
- `read_all` (boolean, optional): read every parameter.
- `privileged` (boolean, optional): run as root - required for writes.
