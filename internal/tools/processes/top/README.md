# processes/top

A one-shot equivalent of `top -b -n 1`, read natively from `/proc` (no `top`/`ps` binary): the summary header followed by the process table with all of top's default columns.

## Output
```text
top - 21:40:01 up  3:02,  1 user,  load average: 0.17, 0.24, 0.52
Tasks: 123 total,   1 running, 122 sleeping,   0 stopped,   0 zombie
%Cpu(s):  1.2 us,  0.5 sy,  0.0 ni, 98.1 id,  0.0 wa,  0.0 hi,  0.2 si,  0.0 st
MiB Mem :   1892.6 total,    437.6 free,    283.5 used,   1171.4 buff/cache
MiB Swap:   3772.0 total,   3772.0 free,      0.0 used.   1413.9 avail Mem

    PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
   3334 root      20   0 2048576 137024  51200 S   1.0   6.9   2:14.07 dockerd
```

| Column | Source |
|---|---|
| `PR`, `NI` | `/proc/<pid>/stat` priority and nice (`rt` for real-time) |
| `VIRT`, `RES`, `SHR` | `/proc/<pid>/statm`, in KiB (scaled to `m`/`g`/`t` when too wide, as top does) |
| `S` | process state |
| `%CPU` | CPU ticks used over the sampling interval - like top, per core, so a multi-threaded process can exceed 100 |
| `%MEM` | `RES` / `MemTotal` |
| `TIME+` | total CPU time, `minutes:seconds.hundredths` |
| `USER` | effective UID |

Header: time; uptime from `/proc/uptime`; logged-in users counted from `/var/run/utmp` (0 inside a container without the host's `/run`); load from `/proc/loadavg`; task counts by state; CPU breakdown from two `/proc/stat` samples; memory and swap from `/proc/meminfo` (`buff/cache` = Buffers + Cached + SReclaimable, like top).

## Parameters
- `sort_by` (string, optional): `cpu` (default), `mem`/`res`, `time`, `pid`.
- `limit` (integer, optional): maximum processes to list; default all.
- `user` (string, optional): only this user's processes.
- `interval_ms` (integer, optional): `%CPU` sampling interval, default 1000, max 10000.
- `output_format` (string, optional): `json`/`yaml`/`table`/`wide` return `{"summary": {...}, "processes": [...]}`, with each process's full `cmdline`, `ppid` and `threads` too.
- `privileged` (boolean, optional): run as root.

`linuxctl get processes top` calls this tool.
