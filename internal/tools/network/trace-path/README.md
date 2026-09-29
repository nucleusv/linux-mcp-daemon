# trace-path

This package implements the `trace-path` tool/resource for the MCP daemon.

## Overview

Traces the network path to a host by running the external `traceroute` binary (not tracepath; it must be installed on the host). Read-only, but it sends probe packets. `host` is a hostname or IP; `max_hops` defaults to traceroute's 30 and is capped at 255. There is no timeout parameter and the 30 s worker limit kills slow traces (30 hops x 3 probes can take minutes), so set a low `max_hops` such as 15. Non-responding hops show as `* * *`. Returns traceroute's raw text; there is no `output_format`. Use `network/ping` first for basic reachability, `network/nslookup` for DNS problems.

## Usage & Permissions

Refer to `configs/mcp-sudo.yaml` to see the default privilege requirements for this feature.
If this tool wraps a privileged binary, the worker execution will run as root if allowed by the configuration.
