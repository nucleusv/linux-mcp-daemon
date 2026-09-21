# nslookup

Query DNS records natively using Go.

This tool resolves DNS queries directly, without invoking `nslookup` or `dig` binaries. It supports `A`, `AAAA`, `TXT`, `MX`, `NS`, and `CNAME` records.

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
