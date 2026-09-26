---
sidebar_label: 'Protocol Methods'
sidebar_position: 1.5
---

# Protocol Methods

Before an AI agent calls a single tool, it asks the daemon what it offers: `initialize`, then `tools/list`, `resources/list` and `resources/templates/list`. Their answers are what the agent sees of mcpd - the tool names, what each does and which arguments it takes - so this page shows them in full, captured live on an Ubuntu 24.04 host. They are sent like any other request, over the SSE session described in the [Overview](./overview); `linuxctl` reads them on every run (`linuxctl get mcp-api tools|resources|info` prints them).

## `initialize`

The first call of a session: the protocol version and what the server implements.

```bash
# POST to the endpoint the SSE stream gave you (see the Overview), then read the reply on the stream
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from the SSE stream>" \
  -H "Authorization: Bearer $MCP_TOKEN" -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "curl", "version": "1"}}}'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "capabilities": {
      "resources": {},
      "tools": {}
    },
    "protocolVersion": "2024-11-05",
    "serverInfo": {
      "name": "linux-mcp-daemon",
      "version": "0.3.3"
    }
  }
}
```

mcpd implements `tools` and `resources` (without subscriptions); `prompts`, `logging` and `completion` aren't declared - see [the `mcp-api` meta-group](../linuxctl/mcp-meta-group) for the details.

## `tools/list`

Every tool the user may call, 38 of them. One entry in full:

```bash
# POST to the endpoint the SSE stream gave you (see the Overview), then read the reply on the stream
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from the SSE stream>" \
  -H "Authorization: Bearer $MCP_TOKEN" -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/list"}'
```

```json
{
  "description": "Returns disk space statistics of the filesystem holding a path, like df - in bytes, or like df -h with human_readable. Use disks/list to see all block devices. (Authorized for 'privileged: true')",
  "inputSchema": {
    "properties": {
      "human_readable": {
        "description": "Sizes like 53.2 GiB (df -h); default is bytes",
        "type": "boolean"
      },
      "inodes": {
        "description": "List inode information instead of block usage (-i)",
        "type": "boolean"
      },
      "output_format": {
        "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
        "type": "string"
      },
      "path": {
        "description": "Absolute path to check",
        "type": "string"
      },
      "privileged": {
        "description": "Set to true to run as root",
        "type": "boolean"
      }
    },
    "required": [
      "path"
    ],
    "type": "object"
  },
  "linuxctl_verb": "free",
  "name": "disks/free",
  "tools_group": "disks"
}
```

| Field | Meaning |
|---|---|
| `name` | What `tools/call` takes: `<group>/<command>` |
| `description` | What the tool does, which tool to use instead for related questions, and - when the user may run it as root - a note saying so |
| `inputSchema` | JSON Schema of the arguments; `required` lists the mandatory ones |
| `tools_group`, `linuxctl_verb` | mcpd's own additions, for `linuxctl`'s `<verb> <group>` grammar; MCP clients ignore them |

<details>
<summary><b>The whole response</b> (38 tools)</summary>

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "tools": [
      {
        "description": "Lists a directory like ls -la: file type and permissions, link count, owner, group, size, modification time and symlink targets (symlinks are shown, never followed). (Hint: You are authorized to run this tool as root. Use 'privileged: true' if you receive permission denied errors on sensitive paths).",
        "inputSchema": {
          "properties": {
            "all": {
              "description": "Include dotfiles, . and .. (ls -a)",
              "type": "boolean"
            },
            "dirs_first": {
              "description": "List directories before files",
              "type": "boolean"
            },
            "human_readable": {
              "description": "Sizes like 4.0K, 1.5M (ls -h); default is bytes",
              "type": "boolean"
            },
            "long": {
              "description": "Long listing like ls -l: type+permissions, links, owner, group, size, date, symlink target. Default true; false lists names only",
              "type": "boolean"
            },
            "numeric_ids": {
              "description": "Show numeric uid/gid instead of names (ls -n)",
              "type": "boolean"
            },
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            },
            "path": {
              "description": "Directory path to list",
              "type": "string"
            },
            "privileged": {
              "description": "Set to true to run as root",
              "type": "boolean"
            },
            "reverse": {
              "description": "Reverse the sort order (ls -r)",
              "type": "boolean"
            },
            "sort": {
              "description": "Sort by name (default), size (largest first) or time (newest first)",
              "enum": [
                "name",
                "size",
                "time"
              ],
              "type": "string"
            }
          },
          "required": [
            "path"
          ],
          "type": "object"
        },
        "linuxctl_verb": "list",
        "name": "files/list",
        "tools_group": "files"
      },
      {
        "description": "Precision reading of file contents with chunking/streaming support.",
        "inputSchema": {
          "properties": {
            "end_line": {
              "description": "Ending line number (inclusive).",
              "type": "integer"
            },
            "limit": {
              "description": "Number of bytes to read.",
              "type": "integer"
            },
            "offset": {
              "description": "Starting byte offset.",
              "type": "integer"
            },
            "path": {
              "description": "Path to the file to read",
              "type": "string"
            },
            "privileged": {
              "description": "Set to true to read as root",
              "type": "boolean"
            },
            "start_line": {
              "description": "Starting line number (1-indexed). Takes precedence over byte offsets.",
              "type": "integer"
            }
          },
          "required": [
            "path"
          ],
          "type": "object"
        },
        "linuxctl_verb": "get",
        "name": "files/read",
        "tools_group": "files"
      },
      {
        "description": "Create a new file or replace file contents.",
        "inputSchema": {
          "properties": {
            "content": {
              "description": "Text content to write to the file",
              "type": "string"
            },
            "path": {
              "description": "Path to the file to create",
              "type": "string"
            },
            "privileged": {
              "description": "Set to true to write as root",
              "type": "boolean"
            }
          },
          "required": [
            "path"
          ],
          "type": "object"
        },
        "linuxctl_verb": "create",
        "name": "files/create",
        "tools_group": "files"
      },
      {
        "description": "Programmatically edit a file by appending text or replacing specific line ranges.",
        "inputSchema": {
          "properties": {
            "append": {
              "description": "If true, appends the content to the end of the file",
              "type": "boolean"
            },
            "content": {
              "description": "Text content to insert or append",
              "type": "string"
            },
            "end_line": {
              "description": "End of the line range to replace (inclusive)",
              "type": "integer"
            },
            "path": {
              "description": "Path to the file to edit",
              "type": "string"
            },
            "privileged": {
              "description": "Set to true to edit as root",
              "type": "boolean"
            },
            "start_line": {
              "description": "Start of the line range to replace (1-indexed)",
              "type": "integer"
            }
          },
          "required": [
            "path",
            "content"
          ],
          "type": "object"
        },
        "linuxctl_verb": "update",
        "name": "files/update",
        "tools_group": "files"
      },
      {
        "description": "Search for files in a directory hierarchy.",
        "inputSchema": {
          "properties": {
            "human_readable": {
              "description": "Sizes like 1.5 KiB; default is bytes",
              "type": "boolean"
            },
            "max_depth": {
              "description": "Maximum depth for directory recursion",
              "type": "integer"
            },
            "mtime": {
              "description": "Modification time (e.g. '+7' for older than 7 days)",
              "type": "string"
            },
            "name": {
              "description": "Glob pattern to match filenames",
              "type": "string"
            },
            "output_format": {
              "description": "Desired output format. Defaults to text",
              "type": "string"
            },
            "path": {
              "description": "Starting directory for the search. Defaults to '/'",
              "type": "string"
            },
            "privileged": {
              "description": "Set to true to search as root",
              "type": "boolean"
            },
            "size": {
              "description": "File size (e.g. '+100M' for larger than 100MB)",
              "type": "string"
            },
            "type": {
              "description": "File type ('f' for file, 'd' for directory, 'l' for symlink)",
              "type": "string"
            }
          },
          "required": [],
          "type": "object"
        },
        "linuxctl_verb": "find",
        "name": "files/find",
        "tools_group": "files"
      },
      {
        "description": "Determines a file's MIME type - the answer `file -b --mime-type` gives, detected natively from the file's first bytes (no file(1) needed). A symlink is reported as inode/symlink, not followed. Use files/stat for size/permissions/ownership instead. (Hint: You are authorized to run this tool as root. Use 'privileged: true' if you receive permission denied errors on sensitive paths).",
        "inputSchema": {
          "properties": {
            "path": {
              "description": "Absolute path to the file",
              "type": "string"
            },
            "privileged": {
              "description": "Set to true to run as root",
              "type": "boolean"
            }
          },
          "required": [
            "path"
          ],
          "type": "object"
        },
        "linuxctl_verb": "filetype",
        "name": "files/filetype",
        "tools_group": "files"
      },
      {
        "description": "Changes a file's or directory's permission bits (chmod). Never follows symbolic links: a path containing a symlink in any component is refused, and recursive changes skip symlinks and report them. Numeric modes follow GNU chmod semantics (on directories a 4-digit mode keeps setuid/setgid; use 5 digits, e.g. 00755, to set them exactly).",
        "inputSchema": {
          "properties": {
            "mode": {
              "description": "Octal (644, 0755, 4755) or symbolic (u+x, go-w, a=r, +X, u+s, +t; comma-separated)",
              "type": "string"
            },
            "path": {
              "description": "Absolute path",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root - needed for files you don't own",
              "type": "boolean"
            },
            "recursive": {
              "description": "Also apply to everything below a directory (symlinks are skipped, never followed)",
              "type": "boolean"
            }
          },
          "required": [
            "path",
            "mode"
          ],
          "type": "object"
        },
        "linuxctl_verb": "chmod",
        "name": "files/chmod",
        "tools_group": "files"
      },
      {
        "description": "Changes a file's or directory's owner and/or group (chown). Never follows symbolic links: a path containing a symlink in any component is refused, and recursive changes skip symlinks and report them. Changing the owner requires privileged: true.",
        "inputSchema": {
          "properties": {
            "owner": {
              "description": "user, user:group, :group, or user: (the user's login group); names or numeric ids",
              "type": "string"
            },
            "path": {
              "description": "Absolute path",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root - required to change ownership",
              "type": "boolean"
            },
            "recursive": {
              "description": "Also apply to everything below a directory (symlinks are skipped, never followed)",
              "type": "boolean"
            }
          },
          "required": [
            "path",
            "owner"
          ],
          "type": "object"
        },
        "linuxctl_verb": "chown",
        "name": "files/chown",
        "tools_group": "files"
      },
      {
        "description": "Returns disk space statistics of the filesystem holding a path, like df - in bytes, or like df -h with human_readable. Use disks/list to see all block devices. (Authorized for 'privileged: true')",
        "inputSchema": {
          "properties": {
            "human_readable": {
              "description": "Sizes like 53.2 GiB (df -h); default is bytes",
              "type": "boolean"
            },
            "inodes": {
              "description": "List inode information instead of block usage (-i)",
              "type": "boolean"
            },
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            },
            "path": {
              "description": "Absolute path to check",
              "type": "string"
            },
            "privileged": {
              "description": "Set to true to run as root",
              "type": "boolean"
            }
          },
          "required": [
            "path"
          ],
          "type": "object"
        },
        "linuxctl_verb": "free",
        "name": "disks/free",
        "tools_group": "disks"
      },
      {
        "description": "Calculates the disk space used by a directory, like du -s - in bytes, or like du -sh with human_readable. Use disks/free for overall partition stats. (Authorized for 'privileged: true' to traverse protected subdirectories)",
        "inputSchema": {
          "properties": {
            "all": {
              "description": "Write counts for all files, not just directories (-a)",
              "type": "boolean"
            },
            "apparent_size": {
              "description": "Print apparent sizes rather than device usage (--apparent-size)",
              "type": "boolean"
            },
            "exclude": {
              "description": "Patterns to exclude",
              "items": {
                "type": "string"
              },
              "type": "array"
            },
            "human_readable": {
              "description": "Sizes like du -h (4.0 KiB); default is bytes",
              "type": "boolean"
            },
            "max_depth": {
              "description": "How deep to recurse (0 for summarize only)",
              "type": "integer"
            },
            "one_file_system": {
              "description": "Skip directories on different file systems (-x)",
              "type": "boolean"
            },
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            },
            "path": {
              "description": "Target directory to measure",
              "type": "string"
            },
            "privileged": {
              "description": "Set to true to run as root",
              "type": "boolean"
            },
            "separate_dirs": {
              "description": "For directories do not include size of subdirectories (-S)",
              "type": "boolean"
            },
            "threshold": {
              "description": "Exclude entries smaller than SIZE if positive, or greater than SIZE if negative (-t)",
              "type": "integer"
            }
          },
          "required": [
            "path"
          ],
          "type": "object"
        },
        "linuxctl_verb": "usage",
        "name": "disks/usage",
        "tools_group": "disks"
      },
      {
        "description": "A snapshot like `top -b -n 1`: header with uptime, logged-in users, load average, task counts by state, CPU breakdown (us/sy/ni/id/wa/hi/si/st) and memory/swap (bytes; MiB with human_readable), followed by the process table with all of top's columns (PID USER PR NI VIRT RES SHR S %CPU %MEM TIME+ COMMAND). %CPU is measured over a short sampling interval, as top does. Use processes/list for a plain listing, processes/delete to signal a process.",
        "inputSchema": {
          "properties": {
            "human_readable": {
              "description": "Memory like top (MiB header, m/g columns); default is bytes",
              "type": "boolean"
            },
            "interval_ms": {
              "description": "%CPU sampling interval in milliseconds (default 1000, max 10000)",
              "type": "integer"
            },
            "limit": {
              "description": "Maximum processes to list (default: all)",
              "type": "integer"
            },
            "output_format": {
              "description": "Default/table: top's own layout. wide: adds PPID, THR and full command lines (like top -c). json/yaml: structured {summary, processes}, memory in bytes (mem_bytes, swap_bytes, virt_bytes, res_bytes, shr_bytes)",
              "type": "string"
            },
            "privileged": {
              "description": "Set to true to run as root",
              "type": "boolean"
            },
            "sort_by": {
              "description": "Sort column: cpu (default, like top), mem/res (resident memory), time (total CPU time), pid",
              "enum": [
                "cpu",
                "mem",
                "res",
                "time",
                "pid"
              ],
              "type": "string"
            },
            "user": {
              "description": "Only this user's processes",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "top",
        "name": "processes/top",
        "tools_group": "processes"
      },
      {
        "description": "Lists running processes on the system. Use this to find a PID, then use the process://{pid}/{target} resource for deep metrics or processes/delete to kill it.",
        "inputSchema": {
          "properties": {
            "human_readable": {
              "description": "RSS like 10Mi; default is bytes",
              "type": "boolean"
            },
            "limit": {
              "description": "Limit returned processes",
              "type": "integer"
            },
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            },
            "pid": {
              "description": "Filter to a single specific PID",
              "type": "integer"
            },
            "privileged": {
              "description": "Set to true to run as root",
              "type": "boolean"
            },
            "sort_by": {
              "description": "Sort by cpu, mem, or pid",
              "type": "string"
            },
            "user": {
              "description": "Filter by username",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "get",
        "name": "processes/list",
        "tools_group": "processes"
      },
      {
        "description": "Terminates a specific process by PID.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            },
            "pid": {
              "description": "The PID to kill",
              "type": "integer"
            },
            "privileged": {
              "description": "Run as root to kill other user's processes",
              "type": "boolean"
            },
            "signal": {
              "description": "Signal to send (e.g., SIGTERM, SIGKILL)",
              "type": "string"
            }
          },
          "required": [
            "pid"
          ],
          "type": "object"
        },
        "linuxctl_verb": "delete",
        "name": "processes/delete",
        "tools_group": "processes"
      },
      {
        "description": "Query DNS records natively.",
        "inputSchema": {
          "properties": {
            "host": {
              "type": "string"
            },
            "record_type": {
              "description": "e.g. A, TXT, MX, CNAME, NS, or ANY",
              "type": "string"
            }
          },
          "required": [
            "host"
          ],
          "type": "object"
        },
        "linuxctl_verb": "nslookup",
        "name": "network/nslookup",
        "tools_group": "network"
      },
      {
        "description": "Transfer data from a URL using native HTTP client.",
        "inputSchema": {
          "properties": {
            "body": {
              "description": "Request body",
              "type": "string"
            },
            "headers": {
              "additionalProperties": {
                "type": "string"
              },
              "description": "Request headers, e.g. {\"Content-Type\": \"application/json\"}",
              "type": "object"
            },
            "insecure": {
              "description": "Skip TLS certificate verification",
              "type": "boolean"
            },
            "max_body": {
              "description": "Return at most this many bytes of the response body (default 1048576 = 1 MiB, max 10 MiB); a cut body has truncated: true",
              "type": "integer"
            },
            "method": {
              "type": "string"
            },
            "timeout": {
              "type": "number"
            },
            "url": {
              "type": "string"
            }
          },
          "required": [
            "url"
          ],
          "type": "object"
        },
        "linuxctl_verb": "curl",
        "name": "network/curl",
        "tools_group": "network"
      },
      {
        "description": "View the system ARP cache (IP to MAC address mappings).",
        "inputSchema": {
          "properties": {
            "interface": {
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "arp",
        "name": "network/arp",
        "tools_group": "network"
      },
      {
        "description": "Measure TCP reachability and latency to a host.",
        "inputSchema": {
          "properties": {
            "host": {
              "type": "string"
            },
            "port": {
              "description": "Defaults to 80",
              "type": "number"
            },
            "timeout": {
              "type": "number"
            }
          },
          "required": [
            "host"
          ],
          "type": "object"
        },
        "linuxctl_verb": "ping",
        "name": "network/ping",
        "tools_group": "network"
      },
      {
        "description": "Lists TCP and UDP sockets in every state with their owning processes, like `ss -tuanp` - read natively from /proc/net (no ss needed). Owning processes of other users' sockets are shown only with privileged: true. state filters by LISTEN (includes unconnected UDP), ESTABLISHED, TIME_WAIT, ... or the groups connected/synchronized. Hint: For physical network links and IPs, use the network://interfaces resource.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            },
            "port": {
              "description": "Filter by port",
              "type": "integer"
            },
            "privileged": {
              "description": "Run as root to see PIDs of other users",
              "type": "boolean"
            },
            "state": {
              "description": "Filter by TCP state, case-insensitive (LISTEN/listening, ESTABLISHED, TIME_WAIT, CLOSE_WAIT, SYN_SENT, ...). Omit to list all sockets - active connections and listening ports.",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "connections",
        "name": "network/connections",
        "tools_group": "network"
      },
      {
        "description": "Returns memory and swap utilization information. Use cpu/load-average to check compute load.",
        "inputSchema": {
          "properties": {
            "detailed": {
              "description": "Set to true to return raw /proc/meminfo instead of summary",
              "type": "boolean"
            },
            "human_readable": {
              "description": "Sizes like free -h (1.8Gi); default is bytes",
              "type": "boolean"
            },
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "usage",
        "name": "memory/usage",
        "tools_group": "memory"
      },
      {
        "description": "Control systemd services (start, stop, restart, enable, disable). To get detailed service properties and state, read the service://{name}/status resource. To view service logs, use the logs/journal-control tool.",
        "inputSchema": {
          "properties": {
            "action": {
              "description": "Action to perform on the service",
              "enum": [
                "start",
                "stop",
                "restart",
                "reload",
                "enable",
                "disable"
              ],
              "type": "string"
            },
            "privileged": {
              "description": "Run as root",
              "type": "boolean"
            },
            "service": {
              "description": "Service name (e.g., 'kubelet.service')",
              "type": "string"
            }
          },
          "required": [
            "service",
            "action"
          ],
          "type": "object"
        },
        "linuxctl_verb": "services",
        "name": "services/manage",
        "tools_group": "system"
      },
      {
        "description": "Lists systemd services with optional filtering. Output includes ActiveState, LoadState, and SubState.",
        "inputSchema": {
          "properties": {
            "active_state": {
              "description": "Filter by active state (e.g., 'active', 'failed', 'inactive')",
              "type": "string"
            },
            "load_state": {
              "description": "Filter by load state (e.g., 'loaded', 'not-found')",
              "type": "string"
            },
            "output_format": {
              "description": "Desired output format (e.g. json, table, wide). Defaults to text",
              "type": "string"
            },
            "pattern": {
              "description": "Wildcard pattern to match service names (e.g., 'kube*', '*ssh*')",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root (may be required depending on policies)",
              "type": "boolean"
            },
            "sub_state": {
              "description": "Filter by sub state (e.g., 'running', 'exited', 'dead')",
              "type": "string"
            }
          },
          "required": [],
          "type": "object"
        },
        "linuxctl_verb": "services",
        "name": "services/list",
        "tools_group": "system"
      },
      {
        "description": "Queries the systemd journal (journalctl equivalent). Requires privileged: true in containerized deployments, since journalctl only exists on the host, never in this daemon's own image.",
        "inputSchema": {
          "properties": {
            "boot": {
              "description": "Restrict output to the current boot (journalctl -b)",
              "type": "boolean"
            },
            "boot_offset": {
              "description": "Select a prior boot relative to the current one, e.g. -1 for the previous boot (implies boot)",
              "type": "integer"
            },
            "lines": {
              "description": "Number of lines to tail (default: 100)",
              "type": "integer"
            },
            "output_format": {
              "description": "Desired output format (e.g. json). Defaults to text",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root and join the host mount namespace - required in containerized deployments",
              "type": "boolean"
            },
            "reverse": {
              "description": "Output newest entries first",
              "type": "boolean"
            },
            "since": {
              "description": "Filter logs since a specific time (e.g., '1 hour ago', 'today')",
              "type": "string"
            },
            "unit": {
              "description": "Filter by systemd unit (e.g., 'kubelet.service')",
              "type": "string"
            },
            "until": {
              "description": "Filter logs until a specific time (e.g., 'yesterday', '12:00')",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "journal",
        "name": "logs/journal-control",
        "tools_group": "logs"
      },
      {
        "description": "Read the kernel ring buffer for hardware/driver logs.",
        "inputSchema": {
          "properties": {
            "level": {
              "description": "Filter by log level (e.g., 'err,warn')",
              "type": "string"
            },
            "output_format": {
              "description": "Output format",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root",
              "type": "boolean"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "dmesg",
        "name": "logs/dmesg",
        "tools_group": "logs"
      },
      {
        "description": "Lists login history (wraps `last`) or failed login attempts (`type: \"failed\"`, wraps `lastb`). Returns raw text, not JSON - last/lastb's output isn't safe to hand-parse into structured data reliably. (Authorized for 'privileged: true' - typically required for type: \"failed\", since btmp is usually root-only readable. When this daemon runs containerized, privileged also automatically reads the real host's login history.)",
        "inputSchema": {
          "properties": {
            "limit": {
              "description": "Only return this many most recent entries",
              "type": "integer"
            },
            "privileged": {
              "description": "Run as root - typically required for type: \"failed\"",
              "type": "boolean"
            },
            "type": {
              "description": "\"success\" (default, wraps `last`) or \"failed\" (wraps `lastb`)",
              "enum": [
                "success",
                "failed"
              ],
              "type": "string"
            },
            "user": {
              "description": "Only return entries for this username",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "logins",
        "name": "logs/logins",
        "tools_group": "logs"
      },
      {
        "description": "Reads or writes kernel parameters (sysctl equivalent) at runtime, natively via /proc/sys. Writes require privileged: true, and may be restricted per user (read-only, or only certain keys) by mcp-sudo.yaml.",
        "inputSchema": {
          "properties": {
            "key": {
              "description": "Kernel parameter name, dotted (net.ipv4.ip_forward) or slash form (net/ipv4/conf/eth0.100/rp_filter). A directory (e.g. net.ipv4) reads its whole subtree.",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root - required for writes",
              "type": "boolean"
            },
            "read_all": {
              "description": "If true, reads all available parameters. Ignored if key is set.",
              "type": "boolean"
            },
            "value": {
              "description": "Value to set for the parameter. If omitted, reads the parameter.",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "sysctl",
        "name": "kernel/system-control",
        "tools_group": "kernel"
      },
      {
        "description": "Retrieves CPU topology and architecture. See cpu/load-average for current utilization.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            },
            "topology_only": {
              "description": "Only return basic core topology",
              "type": "boolean"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "get",
        "name": "cpu/list",
        "tools_group": "cpu"
      },
      {
        "description": "Retrieves system load averages (1m, 5m, 15m). See cpu/list for hardware topology.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "load-average",
        "name": "cpu/load-average",
        "tools_group": "cpu"
      },
      {
        "description": "Lists block devices as a tree (equivalent to lsblk): disks, their partitions, and LVM/dm-crypt/RAID volumes nested under the devices they're built on, with MAJ:MIN, RM, SIZE, RO, TYPE and MOUNTPOINTS. json/yaml output is the same tree under \"blockdevices\" (like lsblk -J), with nested \"children\". To check remaining free space or inode usage, use the disks/free tool. To check which folders are taking up the most space, use the disks/usage tool. (Use 'privileged: true' in containerized deployments to see the host's mount points.)",
        "inputSchema": {
          "properties": {
            "all": {
              "description": "Include empty devices and RAM disks (lsblk -a)",
              "type": "boolean"
            },
            "human_readable": {
              "description": "SIZE like lsblk (60G); default is bytes (lsblk -b)",
              "type": "boolean"
            },
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root - in containerized deployments, reads the host's mount table for MOUNTPOINTS",
              "type": "boolean"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "get",
        "name": "disks/list",
        "tools_group": "disks"
      },
      {
        "description": "Lists mounted filesystems (device, mount point, type, options) - equivalent to `mount`/`findmnt`'s basic view. Use disks/list for block devices instead. (Authorized for 'privileged: true' - when this daemon runs containerized, that automatically shows the real host's mount table, not this container's own.)",
        "inputSchema": {
          "properties": {
            "fs_type": {
              "description": "Only include mounts of this filesystem type (e.g. 'ext4', 'overlay', 'tmpfs')",
              "type": "string"
            },
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root",
              "type": "boolean"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "mounts",
        "name": "disks/mounts",
        "tools_group": "disks"
      },
      {
        "description": "Retrieves granular block device I/O performance metrics (equivalent to iostat). Provides read/write sectors, merged operations, and I/O wait times in milliseconds. Use disks/list first to find valid block devices. If you want static capacity instead, use disks/free.",
        "inputSchema": {
          "properties": {
            "device": {
              "description": "Optional specific block device to query (e.g., 'sda')",
              "type": "string"
            },
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table). Defaults to text",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "performance",
        "name": "disks/performance",
        "tools_group": "disks"
      },
      {
        "description": "Retrieves detailed SMART health data for a drive (equivalent to smartctl -j -a). Returns JSON containing self-assessment test results, temperature, wear leveling, and sector errors. Must be run as root (privileged: true). Use this to diagnose failing hardware.",
        "inputSchema": {
          "properties": {
            "device": {
              "description": "Specific block device to query (e.g., 'sda')",
              "type": "string"
            },
            "privileged": {
              "description": "Run as root - required to read SMART data",
              "type": "boolean"
            }
          },
          "required": [
            "device"
          ],
          "type": "object"
        },
        "linuxctl_verb": "health",
        "name": "disks/health",
        "tools_group": "disks"
      },
      {
        "description": "Retrieves partition boundaries for a drive (start/size, in sectors and bytes), parsed natively from /sys/class/block - no fdisk dependency. Use this to understand the low-level geometry and partition boundaries of a disk.",
        "inputSchema": {
          "properties": {
            "device": {
              "description": "Optional specific block device to query (e.g., 'sda')",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "partitions",
        "name": "disks/partitions",
        "tools_group": "disks"
      },
      {
        "description": "Traces the network path to a host (equivalent to traceroute). Useful for debugging routing issues, identifying where packets are dropped, or measuring network latency across hops. Hint: Use network/ping for basic reachability before tracing the path.",
        "inputSchema": {
          "properties": {
            "host": {
              "description": "Target hostname or IP",
              "type": "string"
            },
            "max_hops": {
              "description": "Maximum number of hops (optional)",
              "type": "integer"
            }
          },
          "required": [
            "host"
          ],
          "type": "object"
        },
        "linuxctl_verb": "trace-path",
        "name": "network/trace-path",
        "tools_group": "network"
      },
      {
        "description": "Retrieves Linux distribution and kernel version.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "os-release",
        "name": "system/os-release",
        "tools_group": "system"
      },
      {
        "description": "Lists installed packages, auto-detecting the package manager (dpkg, apk; rpm-based systems aren't supported natively yet). (Authorized for 'privileged: true' - when this daemon runs containerized, that automatically queries the real host's packages, not this container's own image.)",
        "inputSchema": {
          "properties": {
            "name": {
              "description": "Only packages whose name matches this glob or exact name (e.g. 'openssh-*', '*ssl*', 'curl')",
              "type": "string"
            },
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            },
            "privileged": {
              "description": "Set to true to run as root",
              "type": "boolean"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "packages",
        "name": "system/packages",
        "tools_group": "system"
      },
      {
        "description": "Lists user accounts from /etc/passwd (uid, gid, home, shell, group memberships). Never reads /etc/shadow - this reports account identity, not credentials. (Authorized for 'privileged: true' - when this daemon runs containerized, that automatically lists the real host's users, not this container's own.)",
        "inputSchema": {
          "properties": {
            "min_uid": {
              "description": "Only include users with UID >= this value (e.g. 1000 to exclude system accounts)",
              "type": "integer"
            },
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            },
            "privileged": {
              "description": "Set to true to run as root",
              "type": "boolean"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "get",
        "name": "users/list",
        "tools_group": "users"
      },
      {
        "description": "Returns your authorized tools and privileges from mcp-sudo.yaml.",
        "inputSchema": {
          "properties": {
            "output_format": {
              "description": "Desired output format (e.g. json, yaml, table, wide). Defaults to text",
              "type": "string"
            }
          },
          "type": "object"
        },
        "linuxctl_verb": "sudo-rules",
        "name": "auth/sudo-rules",
        "tools_group": "auth"
      },
      {
        "description": "Re-reads mcpd's config files (daemon.yaml, users.yaml, mcp-sudo.yaml) and applies them without restarting mcpd: users and tokens, per-user grants, rate limits and tool timeouts. It only reads the files - they are edited on the host (linuxctl). They are validated first, strictly (a misspelled key is an error): if any is invalid, nothing changes and the error is returned. Returns what changed (users added/removed, tokens and grants changed). Sessions of removed users, and of users whose token changed, are closed. Server settings (port, TLS, worker.containerized) still need a restart. Only for users granted daemon/reload-config in mcp-sudo.yaml.",
        "inputSchema": {
          "properties": {},
          "type": "object"
        },
        "linuxctl_verb": "reload",
        "name": "daemon/reload-config",
        "tools_group": "daemon"
      }
    ]
  }
}
```

</details>

### The list depends on the user

The response is built per user, from their grants in [`mcp-sudo.yaml`](../configuration/mcp-sudo):

- **Only granted tools are listed where the grant is the permission itself.** `daemon/reload-config` appears only for users granted it: `privileged` sees 38 tools, `unpriviliged` (no grants) sees 37.
- **Descriptions say what the user may run as root.** For a user granted a tool, its description ends with a note like the one above. Without the grant:

```json
{
  "name": "disks/free",
  "description": "Returns disk space statistics of the filesystem holding a path, like df - in bytes, or like df -h with human_readable. Use disks/list to see all block devices."
}
```

So an agent learns from `tools/list` alone where `privileged: true` would be honored - and can't use the list to find out what other users were granted.

## `resources/list`

Fixed resources, read with `resources/read` by their `uri`; 11 of them. One entry:

```bash
# POST to the endpoint the SSE stream gave you (see the Overview), then read the reply on the stream
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from the SSE stream>" \
  -H "Authorization: Bearer $MCP_TOKEN" -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/list"}'
```

```json
{
  "description": "Native system uname information (kernel version, node name). Hint: For CPU hardware architecture use cpu/list tool.",
  "group": "system",
  "linuxctl_verb": "uname",
  "mimeType": "text/plain",
  "name": "OS Uname",
  "uri": "os://uname"
}
```

<details>
<summary><b>The whole response</b> (11 resources)</summary>

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "resources": [
      {
        "description": "Native system uname information (kernel version, node name). Hint: For CPU hardware architecture use cpu/list tool.",
        "group": "system",
        "linuxctl_verb": "uname",
        "mimeType": "text/plain",
        "name": "OS Uname",
        "uri": "os://uname"
      },
      {
        "description": "/etc/os-release information (distribution, version).",
        "group": "system",
        "linuxctl_verb": "release",
        "mimeType": "text/plain",
        "name": "OS Release",
        "uri": "os://release"
      },
      {
        "description": "Native system network hostname. Hint: To resolve IP addresses use network/nslookup tool.",
        "group": "system",
        "linuxctl_verb": "hostname",
        "mimeType": "text/plain",
        "name": "System Hostname",
        "uri": "system://hostname"
      },
      {
        "description": "Configured IANA timezone (e.g. America/New_York) plus current local offset and time.",
        "group": "system",
        "linuxctl_verb": "timezone",
        "mimeType": "text/plain",
        "name": "System Timezone",
        "uri": "system://timezone"
      },
      {
        "description": "Configured locale settings (LANG, LC_*).",
        "group": "system",
        "linuxctl_verb": "locale",
        "mimeType": "text/plain",
        "name": "System Locale",
        "uri": "system://locale"
      },
      {
        "description": "Network interfaces, assigned IP addresses, and detailed RX/TX traffic statistics for all interfaces.",
        "group": "network",
        "linuxctl_verb": "interfaces",
        "mimeType": "application/json",
        "name": "Network Interfaces",
        "uri": "network://interfaces"
      },
      {
        "description": "IPv4 Routing Table (/proc/net/route). Hint: Use network/ping to test reachability.",
        "group": "network",
        "linuxctl_verb": "routes",
        "mimeType": "application/json",
        "name": "Network Routes",
        "uri": "network://routes"
      },
      {
        "description": "Connected USB devices (lsusb equivalent). Lists vendors, products, and bus mapping.",
        "group": "devices",
        "linuxctl_verb": "usb",
        "mimeType": "application/json",
        "name": "USB Devices",
        "uri": "devices://usb"
      },
      {
        "description": "Connected PCI devices (lspci equivalent). Includes network cards, GPUs, and controllers.",
        "group": "devices",
        "linuxctl_verb": "pci",
        "mimeType": "application/json",
        "name": "PCI Devices",
        "uri": "devices://pci"
      },
      {
        "description": "Desktop Management Interface info (lshw/hwinfo equivalent). Detailed hardware specifications (RAM banks, BIOS, chassis).",
        "group": "devices",
        "linuxctl_verb": "dmi",
        "mimeType": "application/json",
        "name": "DMI Hardware Info",
        "uri": "devices://dmi"
      },
      {
        "description": "Loaded kernel drivers (lsmod equivalent). Hint: You can adjust kernel parameters via the kernel/system-control tool.",
        "group": "kernel",
        "linuxctl_verb": "modules",
        "mimeType": "application/json",
        "name": "Kernel Modules",
        "uri": "kernel://modules"
      }
    ]
  }
}
```

</details>

## `resources/templates/list`

Resources with a parameter in the URI (`{pid}`, `{name}`, a path), read with `resources/read` after filling it in; 6 of them. One entry:

```bash
# POST to the endpoint the SSE stream gave you (see the Overview), then read the reply on the stream
curl -s --cacert mcpd.crt -X POST "https://localhost:9091/message?session_id=<from the SSE stream>" \
  -H "Authorization: Bearer $MCP_TOKEN" -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": "1", "method": "resources/templates/list"}'
```

```json
{
  "description": "Reads any file on the system. Append /stat for file metadata, /content for contents, or /type for file type.",
  "group": "files",
  "mimeType": "text/plain",
  "name": "File Reader",
  "uriTemplate": "file:///{path}"
}
```

<details>
<summary><b>The whole response</b> (6 templates)</summary>

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "resourceTemplates": [
      {
        "description": "Reads any file on the system. Append /stat for file metadata, /content for contents, or /type for file type.",
        "group": "files",
        "mimeType": "text/plain",
        "name": "File Reader",
        "uriTemplate": "file:///{path}"
      },
      {
        "description": "Hardware device metadata. Valid types: usb, pci, dmi. Useful for inspecting attached physical hardware.",
        "group": "devices",
        "mimeType": "application/json",
        "name": "Hardware Devices",
        "uriTemplate": "devices://{type}"
      },
      {
        "description": "Detailed properties and RX/TX traffic statistics of a specific network interface.",
        "group": "network",
        "linuxctl_verb": "interfaces",
        "mimeType": "application/json",
        "name": "Network Interface Detail",
        "uriTemplate": "network://interfaces/{name}"
      },
      {
        "description": "Exposes DBus service properties (ActiveState, LoadState, SubState). Best used alongside the services/manage tool to check if a service actually started.",
        "group": "system",
        "linuxctl_verb": "services",
        "mimeType": "application/json",
        "name": "Service Status",
        "uriTemplate": "service://{name}/status"
      },
      {
        "description": "Real-time I/O statistics for a specific block device (e.g. sda). Returns JSON.",
        "group": "disks",
        "mimeType": "application/json",
        "name": "Disk I/O Statistics",
        "uriTemplate": "disks://{name}/stats"
      },
      {
        "description": "Reads process metadata from procfs. Valid targets: status, cmdline, environ, limits (rlimits, e.g. max open files), open_files (fd table - files, sockets, pipes). Hint: Find PIDs using the processes/list tool first.",
        "group": "processes",
        "mimeType": "application/json",
        "name": "Process Introspection",
        "uriTemplate": "process://{pid}/{target}"
      }
    ]
  }
}
```

</details>
