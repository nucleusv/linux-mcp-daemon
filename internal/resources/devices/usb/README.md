# devices://usb

View connected USB devices.

This resource natively reads the `/sys/bus/usb/devices/` virtual filesystem to generate a structured JSON output of all connected USB vendor/product IDs and manufacturer strings, replicating the behavior of `lsusb`. It is executed by an ephemeral isolated worker for security.

## URN

`devices://usb`
