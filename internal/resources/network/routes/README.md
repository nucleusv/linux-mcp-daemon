# Network Routes Resource

This package handles the `network://routes` MCP resource.

## Overview

It parses the IPv4 routing table natively by reading `/proc/net/route`. This allows AI agents to inspect the server's routing policies, default gateways, and network topology without needing the `route` or `ip` CLI tools installed on the host system.

## Output Format
The response is always `application/json`. It returns a JSON Array of Route Objects.

### Route Object
- `iface` (string): The interface the route operates on.
- `destination` (string): The destination network IP.
- `gateway` (string): The gateway IP address.
- `flags` (int): Routing flags.
- `refCnt` (int): Reference count.
- `use` (int): Use count.
- `metric` (int): Routing metric (priority).
- `mask` (string): Network mask.
- `mtu` (int): Maximum Transmission Unit.
- `window` (int): TCP window size.
- `irtt` (int): Initial round trip time.
