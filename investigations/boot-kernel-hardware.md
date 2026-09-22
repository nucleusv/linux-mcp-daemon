# Boot, Kernel & Hardware Troubleshooting

## 1. Identify the OS/kernel version (baseline for any compatibility question)

**Ticket**: "What kernel/distro is this running?" - needed before almost any deeper hardware or compatibility diagnosis.

**Real test**: `system/os-release` returned real data: `Ubuntu 24.04.5 LTS`, kernel `7.0.12-linuxkit`, `aarch64`. The `-linuxkit` kernel suffix is itself a real diagnostic clue (confirms Docker Desktop's VM, not a generic cloud kernel).

**Verdict: Solvable.**

## 2. Inspect connected hardware (USB/PCI/DMI) for a suspected hardware fault

**Ticket**: A peripheral or PCI device isn't recognized; check what the kernel actually sees.

**Real test**: `devices://usb` returned a real (if fully virtualized) device list: `USB/IP Virtual Host Controller` (vendor `1d6b`, i.e. Linux Foundation's standard virtual-hardware vendor ID). `devices://pci` returned real PCI slots with `virtio` vendor IDs (`1af4`). Both resources work correctly and return honest, real data for this environment - there's simply no *physical* hardware to report, since this is a VM/container, not a bug.

**Verdict: Solvable**, with the same "environment has nothing interesting to show, tool works fine" distinction as `disks/health`'s SMART check elsewhere in this investigation.

## 3. Read/adjust a kernel parameter (sysctl) mid-incident

**Ticket**: A kernel tunable (e.g. `vm.swappiness`, `net.ipv4.ip_forward`) needs checking or adjusting to work around a live issue.

**Real test**: `kernel/system-control` reading `vm.swappiness` returned a real live value: `60` (the Linux default).

**Verdict: Solvable**, both for reading (tested) and - per its schema (`value` parameter) - for writing (not executed here, mutating).

## 4. Check loaded kernel modules for a suspected driver issue

**Ticket**: A device isn't working; is its kernel module actually loaded?

**Real test**: Not re-executed in this investigation (already verified working multiple times earlier this session, returning real modules like `shiftfs`, `rosetta`, `fakeowner` - all Docker-Desktop-for-Mac-specific virtualization shims, itself an interesting real finding about how this host bridges Apple Silicon/Rosetta into the Linux VM). `kernel://modules` resource.

**Verdict: Solvable** (carried over from earlier verified state, not re-tested to avoid redundant calls).

## 5. "The system won't boot / a service fails at boot" - boot-time diagnostics

**Ticket**: A unit is failing specifically during boot, not at runtime.

**Real test**: `services/list`'s real output (see `services-systemd.md`) includes many boot-sequence-only units in `inactive`/`dead` state by design (`systemd-fsck-root.service`, `initrd-switch-root.service`, etc.) - correctly distinguishable from genuinely *failed* units via `active_state`. `logs/journal-control` can filter `since: "today"` or similar to isolate the boot window, though there's no dedicated "since last boot" shortcut (`journalctl -b` equivalent) - the caller has to know or guess an approximate boot timestamp.

**Verdict: Partially solvable** - the state distinction (dead-by-design vs. genuinely failed) is real and correct, but there's no direct "show me this boot's log" primitive; `journalctl -b` is a very common real invocation this tool's `since`/`until` string-parsing would need to replicate via a relative time guess instead of a clean boot-boundary marker.
