# Network Routes Resource

This package handles the `network://routes` MCP resource.

## Overview

It parses the IPv4 routing table natively by reading `/proc/net/route`. This allows AI agents to inspect the server's routing policies, default gateways, and network topology without needing the `route` or `ip` CLI tools installed on the host system.

## Output Format
The response is always `application/json`: an array of route objects, decoded from the kernel's raw format (`/proc/net/route` stores addresses as little-endian hex, e.g. `01D4A7DE` = `222.167.212.1`, and flags as a hex bitmask).

```json
[
  {
    "destination": "0.0.0.0/0",
    "gateway": "222.167.212.1",
    "iface": "ens1",
    "metric": 100,
    "flags": ["up", "gateway"],
    "mtu": 0,
    "window": 0,
    "irtt": 0,
    "default": true
  }
]
```

### Route Object
- `destination` (string): Destination network in CIDR notation; `0.0.0.0/0` is the default route.
- `gateway` (string): Next-hop IP, or `""` for directly connected networks.
- `iface` (string): Outgoing interface.
- `metric` (int): Route priority (lower wins).
- `flags` (string[]): Decoded `RTF_*` flags - `up`, `gateway`, `host`, `reinstate`, `dynamic`, `modified`, `reject`.
- `mtu`, `window`, `irtt` (int): Per-route MTU, TCP window and initial RTT overrides (0 = unset).
- `default` (bool): True for the default route.

IPv4 only (`/proc/net/route` has no IPv6 routes).
