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

## Agent Workflow & Rules

To ensure long-term maintainability, AI agents working on this repo must follow these rules:

1. **Mandatory Documentation (READMEs)**
   - Every individual tool (e.g., `internal/tools/disks/performance`) and resource must have its own localized `README.md` explaining how it parses data, what edge cases exist, and any required privileges.
   - Use or write verification scripts (e.g., `scripts/check_readmes.sh`) to automatically find tool directories missing a `README.md`.

2. **When to Update Public Docs**
   - If a new tool requires root privileges, you **must** update `configs/mcp-sudo.yaml` and document it in `docs/website/docs/configuration/mcp-sudo.md`.
   - If you add or rename tools, ensure the `description` in `internal/rpc/tools.go` is rich and explicitly cross-references related tools.

3. **Check Native Replacements First**
   - Before wrapping a bash command (like `lsblk`), always check if a native Go implementation reading `/proc` or `/sys` already exists (like `disks/list`).
