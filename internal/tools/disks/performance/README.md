# disks/iostat

The `disks/iostat` tool retrieves block device I/O statistics by parsing `/proc/diskstats`. It provides the same information as the standard `iostat` command but natively in JSON, YAML, or text format.

## Input Schema

```json
{
  "device": "string (Optional: specific block device to query, e.g. 'sda')",
  "output_format": "string (Optional: 'json', 'yaml', or 'text'. Default: 'text')"
}
```

## Permissions

This tool requires reading `/proc/diskstats`. While typically world-readable, adding it to `mcp-sudo.yaml` ensures the daemon can authorize it explicitly.

## Example

```json
{
  "device": "sda",
  "output_format": "json"
}
```
