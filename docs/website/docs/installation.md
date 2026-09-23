---
sidebar_position: 1.5
sidebar_label: 'Installation'
---

# Installation

Every release is published on [GitHub Releases](https://github.com/nucleusv/linux-mcp-daemon/releases) (Linux amd64/arm64 archives with `mcpd` and `linuxctl`, macOS archives with `linuxctl`, all listed in `checksums.txt`) and as a multi-arch container image, `ghcr.io/nucleusv/linux-mcp-daemon`. Versions follow [SemVer](https://semver.org/); `mcpd --version` and `linuxctl --version` print the installed one.

## Linux with systemd (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | sudo bash
```

The script:

1. downloads the release archive for your architecture (amd64 or arm64) and **verifies its sha256** against the release's `checksums.txt` - a mismatch aborts the install;
2. installs `mcpd` and `linuxctl` to `/usr/local/bin` and the man pages (`man mcpd`, `man linuxctl`);
3. creates `/etc/mcpd/configs/daemon.yaml` and `mcp-sudo.yaml` from clean templates - **no users, no default tokens**;
4. creates one MCP user, `mcp`, backed by an OS account of the same name with no login shell (each tool call runs as that account), and **prints its token once** - only a salted hash is stored;
5. installs the `mcpd` systemd service, enables and starts it.

```text
==> Installed. mcpd listens on port 9091 on all interfaces.

  MCP user:  mcp
  Token:     4c1f...e9a2
             (shown once - only a salted hash is stored; save it now)

  Try it:
    export MCP_SERVER=http://127.0.0.1:9091
    export MCP_TOKEN=4c1f...e9a2
    linuxctl get system os-release
```

### Options

Pass options after `bash -s --`, or as environment variables:

```bash
curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | sudo bash -s -- --version v0.1.0 --user alice
```

| Option | Env | Meaning |
|---|---|---|
| `--version vX.Y.Z` | `MCPD_VERSION` | Install this release instead of the latest |
| `--user NAME` | `MCPD_USER` | Name of the first MCP user (default `mcp`; `""` to create none) |
| `--no-start` | | Install and enable the service, but don't start it |
| `--archive FILE` | | Install from a downloaded release archive (offline hosts) |
| `--uninstall` | | Stop and remove mcpd; keeps `/etc/mcpd` |
| `--uninstall --purge` | | Also delete `/etc/mcpd` |

### Upgrading

Run the same command again. Binaries are replaced (the service is restarted only if `mcpd` actually changed); **existing configs in `/etc/mcpd/configs` are never overwritten**, and no extra user is created.

### Optional system tools

A few tools wrap standard binaries - `disks/health` (`smartctl`), `network/trace-path` (`traceroute`), `files/filetype` (`file`), `network/connections` (`ss`), `logs/journal-control` (`journalctl`). The installer warns about missing ones; on Debian/Ubuntu:

```bash
sudo apt install smartmontools traceroute file iproute2
```

## Container image

```bash
IMAGE=ghcr.io/nucleusv/linux-mcp-daemon:latest   # or a fixed version, e.g. :0.1.0

# 1. Seed a configs directory on the host from the image's clean defaults
sudo mkdir -p /etc/mcpd/configs
sudo docker run --rm -v /etc/mcpd/configs:/out --entrypoint cp $IMAGE -r /root/configs/. /out/

# 2. Create a user and print its token (once)
sudo docker run --rm -v /etc/mcpd/configs:/root/configs $IMAGE \
  linuxctl create mcpd user privileged --config-path /root/configs

# 3. Run, administering the real host
sudo docker run -d --name mcpd --restart unless-stopped \
  --network host --pid host --privileged \
  --mount type=bind,source=/,target=/host,readonly,bind-propagation=rslave \
  -v /etc/mcpd/configs:/root/configs \
  $IMAGE
```

- **MCP users must be OS accounts that exist inside the image** - `testuser`, `unpriviliged` and `privileged` are built in (the name decides which UID each call runs as).
- `--network host --pid host --privileged` plus the read-only host root mount let privileged calls act on the real host (the image's config sets `worker.containerized: true`). Without them, mcpd only administers its own container.
- The image ships **no users and no tokens** - step 2 is required.

## macOS: the CLI

`linuxctl` runs anywhere and talks to a remote mcpd. Download `linuxctl_<version>_darwin_arm64.tar.gz` (Apple Silicon) or `..._darwin_amd64.tar.gz` (Intel) from the [release page](https://github.com/nucleusv/linux-mcp-daemon/releases), then:

```bash
tar xzf linuxctl_*_darwin_*.tar.gz linuxctl
sudo mv linuxctl /usr/local/bin/
export MCP_SERVER=http://your-server:9091 MCP_TOKEN=...
linuxctl get system os-release
```

Shell completion: see [Autocompletion](./linuxctl/autocompletion).

## Before exposing mcpd

- mcpd listens on **all interfaces**, and speaks **plain HTTP** unless TLS is enabled in `daemon.yaml` (`server.tls`). Bearer tokens travel in every request - firewall the port to trusted addresses, or enable TLS, before reaching it over a network.
- Tokens only authenticate; what a user may do **as root** is granted per tool in `mcp-sudo.yaml` - see [mcp-sudo.yaml](./configuration/mcp-sudo). Without grants, a user can call every tool, but only as its own OS account.
- To add more users: `sudo linuxctl create mcpd user NAME --config-path /etc/mcpd/configs`, create the matching OS account, then `sudo systemctl restart mcpd`. See [Daemon User Administration](./linuxctl/mcpd-admin).

## Connecting an AI agent

See [AI Agent Configuration](./ai-agent-configuration) for Claude Code, Claude Desktop and other MCP clients.
