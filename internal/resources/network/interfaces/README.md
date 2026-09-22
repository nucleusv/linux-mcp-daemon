# Network Interfaces Resource

This package handles the `network://interfaces` and `network://interfaces/{name}` MCP resources.

## Overview

It queries the native networking stack via Go's `net.Interfaces()` to return a detailed, structured representation of all active network interfaces and their properties (MAC addresses, MTU, assigned IP addresses, and state flags). 

If a `{name}` is specified in the URI, the results are filtered to only include the interface matching that exact name.

## Output Format
The response is always `application/json`.
- `network://interfaces`: Returns a JSON Array of Interface Objects.
- `network://interfaces/{name}`: Returns a single Interface JSON Object.

### Interface Object
- `index` (int): Interface index.
- `name` (string): Interface name (e.g. `eth0`, `lo`).
- `mac` (string): Hardware MAC address.
- `mtu` (int): Maximum Transmission Unit.
- `flags` (string): Pipe-separated list of state flags (e.g. `up|broadcast|multicast|running`).
- `addresses` (array of string): List of CIDR IP addresses assigned to this interface.
- `statistics` (object): Network traffic statistics (only available on Linux hosts via sysfs). Fields include:
  - `rx_bytes`, `rx_packets`, `rx_errors`, `rx_dropped`
  - `tx_bytes`, `tx_packets`, `tx_errors`, `tx_dropped`
