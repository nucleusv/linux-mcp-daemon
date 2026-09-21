# Kernel Group Implementation Plan

This document tracks the standard Linux kernel manipulation utilities and their implementation status in the MCP Daemon.

## Kernel Modules and Parameters

- [ ] **sysctl**: Configure kernel parameters at runtime (read/write `/proc/sys`).
- [ ] **modprobe / insmod / rmmod**: Add and remove modules from the Linux kernel.
- [x] **lsmod** (`read_modules` resource): Show the status of modules in the Linux kernel.
- [x] **uname -r**: (Covered in `system` group).
