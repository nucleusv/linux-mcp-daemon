# Linux MCP Daemon: Comprehensive Architecture Blueprint

## 1. Project Vision & Philosophy
- **Zero-Dependency (Kernel-First):** No external parsing libraries. Interaction via raw syscalls and VFS (`/proc`, `/sys`).
- **English-Only:** Code, GoDoc, and schemas are in English.
- **HTTP/SSE Transport:** The Master daemon communicates with AI agents (like Claude) directly via HTTP Server-Sent Events (SSE) and JSON-RPC, no bridges required.

## 2. Security & Privilege Separation
- **Master Daemon (Root):** Listens on HTTP/TCP, authenticates via `mcp.passwd`, applies rate limits, and routes requests.
- **Unprivileged Workers:** For safe operations, spawned as the authenticated user.
- **Privileged Workers (Internal Sudo):** For root operations, Master checks `mcp-sudo.yaml` and spawns a single-task ephemeral root process.
- **Rate Limiting:** Token Bucket rate limiter protects against CPU exhaustion.

## 3. Data Pipelines & Caching
- **Singleflight + TTL Cache:** Identical requests are deduplicated.
- **Streaming JSON:** No `bytes.Buffer` allocations for large system responses.