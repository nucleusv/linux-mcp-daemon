# Services Group Implementation Plan

This document tracks the standard Linux service management utilities exposed as resources in the MCP Daemon.

## Service State

- [x] **service status** (`service://{name}/status`): Exposes DBus service properties (ActiveState, LoadState, SubState).
