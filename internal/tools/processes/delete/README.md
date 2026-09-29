# processes/delete

Sends ONE signal to ONE process by PID (kill(2)); the default SIGTERM asks the process to exit. Mutating and not idempotent: it returns as soon as the signal is delivered and does not check that the process exited, and signalling a PID that is gone fails with `no such process`. Allowed `signal` values: SIGTERM, SIGKILL, SIGHUP, SIGINT, SIGQUIT, SIGUSR1, SIGUSR2, SIGSTOP, SIGCONT, SIGABRT (SIG prefix optional, case-insensitive, numbers rejected), so it can also pause (SIGSTOP) and resume (SIGCONT). Refuses PID 1 and the mcpd daemon itself. Another user's process fails with `not permitted` unless `privileged: true` (needs a grant). Find the PID first with `processes/list` or `processes/top`. To stop a managed service use `services/manage` (systemd may restart a killed one), for a container `docker/manage`. Returns one text line, `Successfully sent signal SIGTERM to process N`; `output_format` has no effect.

## Parameters
- `pid` (integer, required), `signal` (string, optional, default `SIGTERM`), `privileged` (boolean, optional).
- `output_format` is accepted but ignored: the reply is always one text line.
