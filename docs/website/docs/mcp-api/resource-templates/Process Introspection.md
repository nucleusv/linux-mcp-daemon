# Process Introspection

**URI Template**: `process://{pid}/{target}`

Reads process metadata from procfs. Valid targets: `status` (state, memory, uid/gid, threads, capabilities), `cmdline` (argv as a JSON array), `environ` (environment as a JSON object), `limits` (rlimits - max open files, max processes, etc. - as a JSON array of `{name, soft, hard, units}`), `open_files` (the process's fd table - files, sockets, pipes - as a JSON array of `{fd, target}`). Hint: Find PIDs using the processes/list tool first.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
linuxctl resource process://1/status
```

</details>

<details>
<summary><b>curl (raw MCP JSON-RPC)</b></summary>

```bash
curl -N -s -H "Authorization: Bearer $MCP_TOKEN" http://localhost:9091/sse &
# server sends: event: endpoint / data: /message?session_id=...

curl -s -X POST "http://localhost:9091/message?session_id=<from step 1>" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/read", "params": {"uri": "process://1/status"}}'
```

</details>

**Real response (captured live):**

```text
Name:	systemd
Umask:	0000
State:	S (sleeping)
Tgid:	1
Ngid:	0
Pid:	1
PPid:	0
TracerPid:	0
Uid:	0	0	0	0
Gid:	0	0	0	0
FDSize:	256
Groups:	0 
NStgid:	1
NSpid:	1
NSpgid:	1
NSsid:	1
Kthread:	0
VmPeak:	   30340 kB
VmSize:	   29784 kB
VmLck:	       0 kB
VmPin:	       0 kB
VmHWM:	   19392 kB
VmRSS:	   19392 kB
RssAnon:	    9480 kB
RssFile:	    9912 kB
RssShmem:	       0 kB
VmData:	    8532 kB
VmStk:	     132 kB
VmExe:	     120 kB
VmLib:	   17976 kB
VmPTE:	      96 kB
VmSwap:	       0 kB
HugetlbPages:	       0 kB
CoreDumping:	0
THP_enabled:	1
untag_mask:	0xffffffffffffff
Threads:	1
SigQ:	0/31882
SigPnd:	0000000000000000
ShdPnd:	0000000000000000
SigBlk:	7fefc1fe28014a03
SigIgn:	0000000000001000
SigCgt:	00000000000004ec
CapInh:	0000000000000000
CapPrm:	000001ffffffffff
CapEff:	000001ffffffffff
CapBnd:	000001ffffffffff
CapAmb:	0000000000000000
NoNewPrivs:	0
Seccomp:	0
Seccomp_filters:	0
Speculation_
...
```
