# System Group Implementation Plan

This document tracks the standard Linux general system utilities and their implementation status in the MCP Daemon.

## General OS and System Information

- [x] **uname / /etc/os-release** (`system/get-os-release`): Print system information, Linux distribution, and kernel version.
- [x] **uptime / /proc/loadavg** (`system/get-load-average`): Display system load averages.
- [x] **systemctl**: Introspect and control the state of the "systemd" system and service manager.
  - Implemented natively via `go-systemd/v22/dbus` API!
  - Tool: `services/manage` (start/stop/enable/disable)
  - Resource: `service://{name}/status`
- [ ] **who / w / users**: Show who is logged on and what they are doing.
- [ ] **last / lastb**: Show listing of last logged in users.
