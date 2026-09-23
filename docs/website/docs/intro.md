---
sidebar_position: 1
---

# Introduction

Welcome to the **Linux MCPd** documentation!

Linux MCPd (`mcpd`) is a high-performance, zero-dependency Go daemon that bridges AI agents to a Linux host over the Model Context Protocol (MCP), via HTTP/SSE + JSON-RPC. It's kernel-first: it parses `/proc`, `/sys`, and DBus natively instead of wrapping CLI tools, and covers far more than the filesystem - files, disks, processes, network, devices, kernel parameters, logs, services, users, CPU, memory, and sudo rules.

## Key Features

- **Kernel-first, zero-dependency**: reads `/proc`/`/sys`/DBus directly rather than shelling out to CLI tools (with a few documented exceptions like `smartctl` and `traceroute`).
- **Secure by design**: an **ephemeral worker** architecture spawns a fresh, short-lived process per call under `syscall.Credential{Uid: targetUID}` - the master daemon loop never touches `/proc`/`/sys` or execs a binary directly.
- **Per-user, per-tool root authorization**: `configs/mcp-sudo.yaml` decides, per bearer-token user and per tool, whether `privileged: true` is honored.
- **Intelligent caching**: `singleflight` deduplication and TTL caching guard against an agent spamming expensive I/O (e.g. recursive disk usage traversals).
- **Salted, hashed bearer tokens**: managed entirely locally via `linuxctl <verb> mcpd user`, never over the network.

## Where to go next

- [Installation](installation) - one command on any systemd Linux host (`curl ... | sudo bash`), the container image, or `linuxctl` for macOS.
- [`linuxctl` CLI](linuxctl/overview) - the schema-discovered `<verb> <group>` command-line client, with a full [command reference](linuxctl/command-reference).
- [MCP API reference](mcp-api/overview) - every tool, resource, and resource template `mcpd` exposes, each with a runnable `linuxctl` + raw `curl` example and real captured output.
- [Architecture](architecture/ephemeral-workers) and [Configuration](configuration/daemon) - the worker/privilege model and `configs/*.yaml` reference.
