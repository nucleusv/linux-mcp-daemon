# network/arp

Shows the kernel's ARP cache (IPv4 address to MAC address) from /proc/net/arp in the daemon's network namespace. Read-only. It is a cache, not a scan: only hosts contacted recently appear, and IPv6 neighbours are not included. `interface` is an exact device name (`eth0`); omit it for all. For sockets and connections use `network/connections`, for reachability `network/ping`. Always returns JSON: an array of objects (ip_address, hw_type and flags as raw hex such as `0x1`, hw_address, mask, device). When nothing matches the output is `null`, not `[]`.

This tool natively reads `/proc/net/arp` and constructs a JSON output of the cache mappings, avoiding the need for `arp` or `ip neigh` binaries.

## Input Schema

```json
{
  "type": "object",
  "properties": {
    "interface": {
      "type": "string",
      "description": "Optional interface name to filter the results."
    }
  }
}
```
