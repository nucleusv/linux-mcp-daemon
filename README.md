# Linux MCP Daemon

A high-performance, Go-based Model Context Protocol (MCP) daemon designed to securely bridge AI agents directly with the Linux operating system.

## Overview

This project implements a zero-dependency (kernel-first) philosophy. It allows AI agents to introspect and interact with the host Linux system directly via raw syscalls and the Virtual File System (`/proc`, `/sys`) without requiring bloated third-party parsing libraries.

Communication happens directly between the AI agent and the daemon via HTTP Server-Sent Events (SSE) and JSON-RPC over port 9090.

## Features

- **Direct AI Interaction:** HTTP/SSE transport for immediate agent-to-daemon communication.
- **Strict Security:** Rate limiting, Bearer token authentication, and directory traversal protection.
- **Privilege Separation:** Master daemon runs as root, spinning up ephemeral unprivileged/privileged workers based on rules defined in `configs/mcp-sudo.yaml`.
- **High Performance:** Uses `singleflight` deduplication and TTL caching for efficient system introspection.
- **Docker & Kubernetes Ready:** Fully containerized with a multi-stage Docker build and Kubernetes deployment manifests that allow safe host introspection.

## Getting Started

### 1. Build and Deploy the Server (`mcpd`)

1. Build the Docker image:
   ```bash
   ./scripts/build.sh
   ```

2. Deploy to your local Kubernetes cluster:
   ```bash
   ./scripts/deploy.sh
   ```

### 2. Build the Client (`linuxctl`)

You can build the CLI client directly on your host machine (e.g. macOS):

```bash
./scripts/build-cli.sh
```

### 3. Usage

The daemon runs on port `9090`. You can connect via your AI client using SSE, or use the `linuxctl` CLI tool:

```bash
# Set your token as an environment variable
export MCP_TOKEN="your_token_here"

# Ping the daemon
./linuxctl ping
```

## Project Structure

- `cmd/mcpd/`: Main server application entrypoint.
- `cmd/linuxctl/`: The CLI client application.
- `configs/`: Configuration files (Daemon config, Sudo rules).
- `internal/`: Encapsulated business logic:
  - `auth/`: Authentication, authorization, rate limiting.
  - `fs/`: Safe, atomic file system operations.
  - `kernel/`: Raw parsing of `/proc` and `/sys`.
  - `mcpcore/`: Core routing and caching logic.
- `k8s/`: Kubernetes deployment manifests.
- `scripts/`: Build and deployment automation.

## License

This project is licensed under the MIT License.
