---
sidebar_position: 1
---

# Introduction

Welcome to the **Linux MCP Daemon** documentation!

The Linux MCP Daemon is a high-performance, ultra-secure Model Context Protocol (MCP) server written in Go. It empowers Large Language Models (LLMs) to safely interact with a Linux filesystem.

## Key Features

- **Blazing Fast**: Written in pure Go, taking advantage of fast static compilation and low memory overhead.
- **Secure by Design**: Utilizes an **Ephemeral Worker** architecture to completely sandbox AI requests under strict UID controls, preventing privilege escalation.
- **Intelligent Caching**: Hardened against abuse with `golang.org/x/sync/singleflight` to prevent the AI from spamming expensive I/O operations like recursive disk usage tree traversals.
- **Dynamic Sudo Rules**: Allows administrators to grant specific users granular root access (`privileged: true`) for specific tools via a `mcp-sudo.yaml` configuration.

Explore the sidebar to dive deep into the specific tools (like `list_directory` or `get_disk_space`) or read up on our architectural design choices!
