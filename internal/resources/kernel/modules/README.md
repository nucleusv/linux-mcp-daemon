# kernel://modules

View loaded kernel drivers.

This resource natively parses `/proc/modules` to list all dynamically loaded kernel modules in memory, replicating the behavior of `lsmod`. It is executed by an ephemeral isolated worker for security.

## URN

`kernel://modules`
