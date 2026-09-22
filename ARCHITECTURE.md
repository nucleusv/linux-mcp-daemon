# Architecture & Agent Context

This document serves as persistent memory for AI agents interacting with the `linux-mcp-daemon-by-antigravity` repository.

## Core Philosophy
- **Kernel-First, Zero-Dependency**: We avoid wrapping messy external CLI binaries unless absolutely necessary. We prefer parsing native Linux structures (like `/proc` and `/sys`).
- **Privilege Separation**: The daemon runs as root, but spawns ephemeral workers to execute tasks. Permissions are strictly governed by `configs/mcp-sudo.yaml`.

## Known Tools & Resources Mapping

### Disks & Storage
- **`disks/list` (Tool)**: Lists block devices, partitions, and their trees natively by parsing `/proc/partitions` and `/sys/block`. **Do not attempt to implement `lsblk`**, as this tool already natively replaces it.
- **`disks/iostat` (Tool)**: Parses `/proc/diskstats` for granular I/O metrics.
- **`disks://{name}/stats` (Resource Template)**: Exposes the `disks/iostat` tool as an instantiated JSON resource for a specific block device.
- **`disks/fdisk` & `disks/smartctl` (Tools)**: These require root execution via `privileged: true` and execute external binaries (`fdisk -l` and `smartctl -j -a`) because reading partition tables and SMART data directly from raw block devices in Go is too complex and brittle.

### Network
- **`network/traceroute` (Tool)**: Native wrapper around the `traceroute` binary.

### Services
- We use `go-systemd/v22/dbus` for native systemd service management.

## Testing Guidelines
- All tests **MUST** be written via the MCP JSON-RPC interface, not via raw HTTP/Curl requests against the server.
