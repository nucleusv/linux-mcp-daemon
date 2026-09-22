---
sidebar_label: 'CLI / linuxctl'
---

# linuxctl CLI

The `linuxctl` command-line interface allows users and AI agents to seamlessly interact with the Linux Model Context Protocol (MCP) Daemon (`mcpd`) running in the background.

## Synopsis

```bash
linuxctl [OPTIONS] resources
linuxctl [OPTIONS] resource <uri>
linuxctl [OPTIONS] <group>/<command> [command-options]
```

## Options

- `-server URL`
  The URL of the `mcpd` server to connect to. Defaults to `http://localhost:9090`.

- `-token TOKEN`
  Bearer token for authentication. If not provided via this flag, the client will automatically look for the `MCP_TOKEN` environment variable.

## Commands

### `ping`
Pings the `mcpd` daemon to verify connectivity and authentication. Returns a success message if the daemon is reachable and the token is valid.

### `resources`
Fetches and lists all statically available resources and resource templates exposed by the daemon.

### `resource <uri>`
Reads the exact contents of an MCP resource (e.g. `file:///etc/hosts` or `os/hostname`). 
Use `--output table` or `--output json` to optionally format structural JSON outputs.

### `<group>/<command>`
Dynamically executes an MCP tool exposed by the daemon. For example, `files/list` or `disks/free`. Run `linuxctl` without arguments to see available groups, or `linuxctl <group>` to see available commands within a specific group.

### `[command-options]`
Arguments specific to the command being executed. You can provide these either positionally:
```bash
linuxctl files/list /var/log
```
Or as explicit flags:
```bash
linuxctl files/list --path /var/log --privileged true
```
You can also append `--output table` or `--output json` to format structural data returned by tools.

## Environment Variables

- `MCP_TOKEN`
  The bearer token used for authenticating with the daemon. This is the recommended way to authenticate so you don't leak tokens in shell histories.

## Examples

**Ping the daemon using a specific token:**
```bash
linuxctl -token "my-test-token-123" ping
```

**Ping the daemon using an environment variable (Recommended):**
```bash
export MCP_TOKEN="my-test-token-123"
linuxctl ping
```

**List all available resources:**
```bash
linuxctl resources
```

**Read the system uname as a resource:**
```bash
linuxctl resource os/release
```

**List files in a directory:**
```bash
linuxctl files/list --path /var/log
```

**Run a privileged tool (requires root rule in `configs/mcp-sudo.yaml`):**
```bash
linuxctl disks/free --privileged true
```
