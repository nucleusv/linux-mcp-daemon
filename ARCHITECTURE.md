# Architecture & Agent Context

This document serves as persistent memory for AI agents interacting with the `linux-mcp-daemon-by-antigravity` repository.

## Core Philosophy
- **Kernel-First, Zero-Dependency**: We avoid wrapping messy external CLI binaries unless absolutely necessary. We prefer parsing native Linux structures (like `/proc` and `/sys`).
- **Privilege Separation**: The daemon runs as root, but spawns ephemeral workers to execute tasks. Permissions are strictly governed by `configs/mcp-sudo.yaml`.

## Gotcha: privileged resource reads need TWO grants, not one
A resource template (`service://{name}/status`, `file://{path}`, `process://{pid}/{target}`) computes its own `isPrivileged` via `SudoConfig.CanReadResourceAsRoot(user, scheme, value)` - gated by the `resources:` block in `mcp-sudo.yaml` (e.g. `resources: service://: [""]`). But `SpawnWorker` *independently* re-checks `CanRunAsRoot(user, toolName)` using the internal worker name behind that resource (`services/status`, `files/content`, `processes/read`) - gated by the `tools:` block. Both checks must pass; granting only the `resources:` entry produces a confusing "not authorized to use privileged: true" error that looks like a tools-side problem even though the resource-side grant is correct. These internal worker names are never valid `tools/call` targets (they don't appear in `tools.go`'s schema/`standardWorkers`), so it's easy to mistake their `tools:` entry as dead config and remove it - it isn't dead if anything grants privileged access to the resource that uses it. Exception: `read_usb`/`read_pci`/`read_dmi`/`read_modules`/`read_routes` are exempted from the `CanRunAsRoot` check entirely (see the `strings.HasPrefix(toolName, "read_")` special case in `spawner.go`) since the resource-level check already fully gates them - only `services/status`/`files/content`/`processes/read` need this dual grant, because they don't follow that naming convention.

Also note for prefix-matched `resources:` entries (`file://`, `service://`, `process://`): "grant everything" is an empty-string prefix (`- ""`), **not** `"*"`. `"*"` is only meaningful for the small set of *exact-match* resources (`devices://usb`, `kernel://modules`, etc.), where the code always compares against the literal sentinel string `"*"` rather than doing prefix matching - see `CanReadResourceAsRoot` in `internal/config/sudo.go`.

## Host Filesystem Access (automatic, via `worker.containerized`)
When the daemon itself runs containerized (its actual deployment mode - see `k8s/deployment.yaml`), a worker's normal filesystem view is the daemon's own container image, not the real host, even with `privileged: true` (which only elevates to root *inside that container*). `internal/worker/JoinHostMountNamespace` (`internal/worker/hostns.go`) gives a worker genuine host filesystem access instead: it calls `setns(CLONE_NEWNS)` on `/proc/1/ns/mnt` (PID 1 is the host's real init, reachable because the pod sets `hostPID: true`) and then `chroot`s into `/proc/1/root`. No external `nsenter` binary or new Go dependency - `syscall.SYS_SETNS`/`syscall.CLONE_NEWNS` are already in the standard library, consistent with the zero-dependency philosophy above.

**This is a daemon-wide setting, not a per-call tool argument.** `configs/daemon.yaml`'s `worker.containerized: true` is read once at startup into `worker.Containerized` (a package-level var); `SpawnWorker` then calls `JoinHostMountNamespace()` for *every* `privileged: true` call automatically, with no separate flag for a caller to know about or set. This was a deliberate correction of an earlier design: a first pass exposed a per-tool `"host_root": true` JSON argument, but that fails for two reasons worth remembering if the idea resurfaces - (1) nothing in this daemon's actual usage wants "root inside the container only" as *distinct* from "root on the real host", so the extra flag was a distinction nobody needed to make per call, and (2) an AI/MCP client can only discover parameters that appear in a tool's `inputSchema`; a resource (`service://`, `file://`, `process://`) has no equivalent schema for its URI, so a resource-side flag would have been fundamentally undiscoverable to any client, structurally, not just as an oversight. Folding the decision into daemon startup config fixes both: no new parameter for any tool or client to learn, and resources inherit correct host access for free (they already compute their own `isPrivileged` and pass it into `SpawnWorker`, same as tools).

**Gotcha that cost real debugging time**: Go's runtime OS threads share `fs_struct` (`CLONE_FS`) by default, and the kernel refuses `setns(CLONE_NEWNS)` (`EINVAL`) on any thread that shares filesystem attributes with others. `JoinHostMountNamespace` must call `syscall.Unshare(syscall.CLONE_FS)` on the locked OS thread before `setns` - without it, every call fails with `setns(CLONE_NEWNS): invalid argument`, which looks like a permissions or argument-ordering bug but isn't.

**Safety net**: `JoinHostMountNamespace` first compares `/proc/self/ns/mnt` against `/proc/1/ns/mnt` (device+inode, via the `sameNamespace` helper) and no-ops if they already match - so a misconfigured `worker.containerized: true` on a non-containerized host doesn't demand `CAP_SYS_ADMIN` for nothing. The same comparison, exposed as `worker.IsContainerized()`, drives a startup warning in `cmd/mcpd/main.go` if the configured `worker.containerized` value doesn't match what the process can actually detect about itself.

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

### System
- **`system/packages` (Tool)**: Lists installed packages by natively parsing `/var/lib/dpkg/status` (Debian/Ubuntu) or `/lib/apk/db/installed` (Alpine) - both are plain-text, safe to hand-parse. RPM-based systems are **not** supported: RPM's package database is a Berkeley DB / SQLite file, not something to hand-parse without a real library, so this is a documented gap rather than a `rpm -qa` wrapper - add one deliberately if RPM support is ever needed, following the `smartctl`/`traceroute` precedent above. With `privileged: true`, automatically reports on the real host rather than this container's own image (see "Host Filesystem Access" above).

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
   - **Whenever you modify, add, or rename tools/resources in the registries (`tools.go` or `resources.go`), you MUST regenerate the Docusaurus website documentation by running:**
     ```bash
     python3 scripts/generate_docs.py
     ```

3. **Check Native Replacements First**
   - Before wrapping a bash command (like `lsblk`), always check if a native Go implementation reading `/proc` or `/sys` already exists (like `disks/list`).
