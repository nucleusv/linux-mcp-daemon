# network/nslookup

Resolves DNS records for a hostname through the host's resolver (/etc/resolv.conf; a DNS server cannot be chosen), so answers may come from a local cache. Read-only, but it sends queries off-host. `record_type` (case-insensitive) is A, AAAA, CNAME, TXT, MX, NS or ANY; the default ANY is not a DNS ANY query but runs the CNAME, A/AAAA, TXT, MX and NS lookups in turn. Other types (SOA, PTR, SRV) are rejected, and `host` must be a name: an IP address is not reverse-resolved. Always returns JSON: an object with `host` and `records` (array of objects with `type` and `value`; an MX value looks like `10 mail.example.com.`). Empty `records` means the name exists but has no record of that type; a name that does not exist is an error (`HOST: no such host`). To test reachability use `network/ping`, to fetch a URL `network/curl`.

This tool resolves DNS queries directly, without invoking `nslookup` or `dig` binaries. It supports `A`, `AAAA`, `TXT`, `MX`, `NS`, and `CNAME` records, or `ANY` (the default), which runs all of these lookups in turn.

## Input Schema

```json
{
  "type": "object",
  "properties": {
    "host": {
      "type": "string"
    },
    "record_type": {
      "type": "string",
      "description": "Optional record type to query. Valid options: A, TXT, MX, CNAME, NS, or ANY. Defaults to ANY."
    }
  },
  "required": ["host"]
}
```
