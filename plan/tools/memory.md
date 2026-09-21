# Memory Group Implementation Plan

This document tracks the standard Linux memory utilities and their implementation status in the MCP Daemon.

## Memory and Swap

- [x] **free / /proc/meminfo** (`memory/get-memory`): Display amount of free and used memory in the system.
- [ ] **vmstat**: Report virtual memory statistics.
- [ ] **smem**: Report memory usage with shared memory divided proportionally.
- [ ] **pmap**: Report memory map of a process (could be part of `processes` or `memory`).
