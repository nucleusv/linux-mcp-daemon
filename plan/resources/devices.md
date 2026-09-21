# Devices Group Implementation Plan

This document tracks the standard Linux hardware and device utilities and their implementation status in the MCP Daemon.

## Hardware Information

- [x] **lsusb**: List USB devices (`read_usb` resource).
- [x] **lspci**: List all PCI devices (`read_pci` resource).
- [ ] **lshw**: Extract detailed information on the hardware configuration of the machine.
- [x] **lsmod**: Show the status of modules in the Linux kernel (`read_modules` resource).
- [x] **hwinfo**: Hardware identification system (Implemented DMI decoding via `read_dmi` resource).
