# System Group Implementation Plan

This document tracks the standard Linux general system utilities and their implementation status in the MCP Daemon.

## General OS and System Information

- [x] **uname / /etc/os-release** (`system/get-os-release`): Print system information, Linux distribution, and kernel version.
- [x] **uptime / /proc/loadavg** (`system/get-load-average`): Display system load averages.
- [ ] **systemctl**: Control the systemd system and service manager.
- [ ] **who / w / users**: Show who is logged on and what they are doing.
- [ ] **last / lastb**: Show listing of last logged in users.
