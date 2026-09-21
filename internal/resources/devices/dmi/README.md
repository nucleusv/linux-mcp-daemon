# devices://dmi

View Desktop Management Interface (DMI) hardware info.

This resource natively reads the `/sys/class/dmi/id/` virtual filesystem to pull static hardware information such as the system vendor, product name, and BIOS version, replicating the behavior of `lshw` or `hwinfo`. It is executed by an ephemeral isolated worker for security.

## URN

`devices://dmi`
