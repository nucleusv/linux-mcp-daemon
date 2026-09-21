# devices://pci

View connected PCI devices.

This resource natively reads the `/sys/bus/pci/devices/` virtual filesystem to generate a structured JSON output of all connected PCI slots, classes, and vendor/device IDs, replicating the behavior of `lspci`. It is executed by an ephemeral isolated worker for security.

## URN

`devices://pci`
