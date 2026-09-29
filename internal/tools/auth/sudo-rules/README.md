# auth/sudo-rules

Shows which tools YOU may run as root: your `privileged` grants from mcp-sudo.yaml with their `paths`, `containers`, `prune`, `network` and `sysctl` restrictions. It does not list which tools you may call at all: unprivileged calls need no grant, except docker/* and daemon/reload-config, which always do. Read-only and answered by the daemon itself. Call it before a `privileged: true` request or after a permission-denied error. Text is `Your authorized privileged tools:` followed by JSON; `output_format: json` returns only that JSON, whose keys are capitalised Go field names (Tools, Resources, Allowed, Paths, Containers, Prune, Network, Sysctl). No grants gives `You have no privileged tools authorized in mcp-sudo.yaml.` (JSON: `{}`). After an operator edits grants, `daemon/reload-config` applies them.

## Standard Output Format
All tools in the Linux MCP Daemon natively support returning structured JSON output. This can be requested by passing the `output_format` parameter.

## Parameters
- `output_format` (string, optional): The requested format. Setting this to `json`, `yaml`, `table`, or `wide` will return the raw structured JSON payload instead of human-readable text.
