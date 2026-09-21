# arp

View the system ARP cache (IP to MAC address mappings).

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
