# Process Introspection

**URI Template**: `process://{pid}/{target}`

Reads process metadata from procfs. Valid targets: `status` (state, memory, uid/gid, threads, capabilities), `cmdline` (argv as a JSON array), `environ` (environment as a JSON object), `limits` (rlimits - max open files, max processes, etc. - as a JSON array of `{name, soft, hard, units}`), `open_files` (the process's fd table - files, sockets, pipes - as a JSON array of `{fd, target}`). Hint: Find PIDs using the processes/list tool first.
