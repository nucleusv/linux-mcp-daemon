# Journal-Control

**Tool Name**: `logs/journal-control`

Queries the systemd journal (`journalctl` equivalent).

## Parameters

- `unit` (optional) - filter by systemd unit (e.g. `kubelet.service`).
- `lines` (optional) - number of most recent lines to return. Defaults to 100.
- `since` / `until` (optional) - time range filters (e.g. `"1 hour ago"`, `"yesterday"`, `"12:00"`).
- `reverse` (optional) - newest entries first.
- `boot` (optional) - restrict to the current boot (`journalctl -b`).
- `boot_offset` (optional) - select a prior boot relative to the current one, e.g. `-1` for the previous boot; implies `boot`.
- `output_format` (optional) - `json` for structured journal entries; omit for plain text.
- `privileged` (optional, **required in containerized deployments**) - run as root and join the host mount namespace. `journalctl` and the journal it reads are architecturally host-only and are never bundled into this daemon's own container image; without `privileged: true` this call fails with `journalctl: executable file not found in $PATH`.

## Example

Every example below shows the equivalent `linuxctl` command and the raw MCP JSON-RPC call it resolves to. The raw call always follows the same two-step pattern (see [MCP API overview](../../overview) for the full explanation): open an SSE stream to get a one-time POST endpoint, then POST the JSON-RPC request there - the result streams back on the SSE connection.

<details>
<summary><b>linuxctl</b></summary>

```bash
$ linuxctl get logs journal --unit kubelet.service --lines 3 --privileged true
Sep 22 23:03:19 desktop-control-plane kubelet[268]: I0922 23:03:19.328923 ...
Sep 22 23:03:15 desktop-control-plane kubelet[268]: I0922 23:03:15.316812 ...
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
  -d '{"jsonrpc": "2.0", "id": "1", "method": "tools/call", "params": {"name": "logs/journal-control", "arguments": {"unit": "kubelet.service", "lines": 3, "privileged": true}}}'
```

</details>

**Real response (captured live):**

```text
Sep 23 11:40:47 desktop-control-plane kubelet[471811]: I0923 11:40:47.863211  471811 kubelet_volumes.go:161] "Cleaned up orphaned pod volumes dir" podUID="988275a3-df6f-4a3d-a432-8ef6676b2cce" path="/var/lib/kubelet/pods/988275a3-df6f-4a3d-a432-8ef6676b2cce/volumes"
Sep 23 11:41:13 desktop-control-plane kubelet[471811]: I0923 11:41:13.979360  471811 pod_startup_latency_tracker.go:148] "Observed pod startup duration" pod="linux-mcp-daemon-by-claude/linux-mcp-daemon-8cdfd76f-mwwhz" podStartSLOduration=26.79312747 podStartE2EDuration="26.973310511s" totalImagesPullingTime="180.183041ms" totalInitContainerRuntime="0s" isStatefulPod=true podCreationTimestamp="2026-09-23 11:40:47 +0000 UTC" imagePullSessionsCount=1 imagePullSessionsStartsCount=0 observedRunningTime="2026-09-23 11:40:48.09362143 +0000 UTC m=+4208.644264215" watchObservedRunningTime="2026-09-23 11:41:13.973310511 +0000 UTC m=+42
...
```
