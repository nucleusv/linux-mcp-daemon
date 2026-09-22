# Processes Group Implementation Plan

This document tracks the standard Linux process metadata exposed as resources in the MCP Daemon.

## Process Introspection (procfs)

- [x] **process status** (`process://{pid}/status`): Expose the `/proc/[pid]/status` file (memory, thread count, state).
- [x] **process cmdline** (`process://{pid}/cmdline`): Expose the raw command line of the process.
- [x] **process environ** (`process://{pid}/environ`): Expose the environment variables of the process.
