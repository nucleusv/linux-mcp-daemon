# Disks Group Implementation Plan

This document tracks the standard Linux disk utilities and their implementation status in the MCP Daemon.

## Block Devices and Filesystems

- [x] **df** (`disks/disk-free`): Report file system disk space usage.
- [x] **du** (`disks/disk-usage`): Estimate file space usage.
- [x] **lsblk** (`disks/list-blocks`): List block devices.
- [ ] **findmnt / mount**: Find a filesystem mount, or list mounted filesystems.
- [ ] **fdisk / parted**: Manipulate disk partition table (view only, probably).
- [ ] **blkid**: Locate/print block device attributes.
- [ ] **smartctl**: Control and monitor utility for SMART disks.
- [ ] **iostat**: Report Central Processing Unit (CPU) statistics and input/output statistics for devices and partitions.
- [ ] **lsof**: List open files.
