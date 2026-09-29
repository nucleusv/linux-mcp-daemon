# disks/performance

Returns block-device I/O counters from /proc/diskstats: reads and writes completed and merged, sectors (512 bytes) and milliseconds spent, in-flight I/Os and weighted I/O time. Values are CUMULATIVE since boot, not rates and not iostat's per-interval figures; there is no %util or await, so sample twice and subtract to get a rate. Read-only. Without `device` all devices are listed except `loop*` and `ram*`; a name such as `sda` (find them with `disks/list`) selects one, and an unknown name gives `no such block device`. Text output is a table; `output_format: json` and `yaml` (real YAML here) return an array of objects with 14 fields (major, minor, device_name, reads_completed, ..., weighted_time_ios_ms). For capacity use `disks/free`, for SMART health `disks/health`.

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
