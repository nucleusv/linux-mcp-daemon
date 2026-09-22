# health

**Tool Name**: `disks/health`

Retrieves detailed SMART health data for a drive (equivalent to smartctl -j -a). Returns JSON containing self-assessment test results, temperature, wear leveling, and sector errors. Must be run as root (privileged: true). Use this to diagnose failing hardware.
