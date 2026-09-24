---
sidebar_position: 1.5
sidebar_label: 'Installation'
---

# Installation

Every release is published on [GitHub Releases](https://github.com/nucleusv/linux-mcp-daemon/releases) (Linux amd64/arm64 archives with `mcpd` and `linuxctl`, macOS archives with `linuxctl`, all listed in `checksums.txt`) and as a multi-arch container image, `ghcr.io/nucleusv/linux-mcp-daemon`. Versions follow [SemVer](https://semver.org/); `mcpd --version` and `linuxctl --version` print the installed one.

## Linux with systemd (recommended)

You need a Linux host (amd64 or arm64), root access via `sudo`, and `curl`. Minimal images such as the `ubuntu`/`debian` containers ship neither `curl` nor `sudo`, and have empty package lists: as root there, run `apt update && apt install -y curl ca-certificates` and pipe the script to plain `bash` instead of `sudo bash`.

```bash
curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | sudo bash
```

The script:

1. downloads the release archive for your architecture (amd64 or arm64) and **verifies its sha256** against the release's `checksums.txt` - a mismatch aborts the install;
2. installs `mcpd` and `linuxctl` to `/usr/local/bin` and the man pages (`man mcpd`, `man linuxctl`);
3. creates `/etc/mcpd/configs/daemon.yaml`, `users.yaml` and `mcp-sudo.yaml` from clean templates - **no users, no default tokens**;
4. creates one MCP user, `mcp`, backed by an OS account of the same name with no login shell (each tool call runs as that account), and **prints its token once** - only a salted hash is stored. `mcp` may also apply config changes ([`daemon/reload-config`](./mcp-api/tools/daemon/reload-config)), so users you add later take effect without a restart;
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

Run those three commands (with your token) as any user on the host - `linuxctl` talks to mcpd over HTTP, so no `sudo` is needed. `linuxctl get processes top` is another quick check.

**Without systemd** (e.g. inside a plain container) the script installs everything except the service, warns `systemd not detected`, and prints how to start mcpd by hand: `cd /etc/mcpd && sudo /usr/local/bin/mcpd`. It runs in the foreground - use a second terminal for "Try it", or start it with `&` in the background. Nothing restarts or stops it for you: after an upgrade restart it yourself, and `--uninstall` only warns that it is still running.

### Options

Pass options after `bash -s --`:

```bash
curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | sudo bash -s -- --version v0.1.0 --user alice
```

or as environment variables - put them **after `sudo`**, since `sudo` drops variables exported in your shell (`export MCPD_USER=alice` beforehand is silently ignored):

```bash
curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | sudo MCPD_VERSION=v0.1.0 MCPD_USER=alice bash
```

| Option | Env | Meaning |
|---|---|---|
| `--version vX.Y.Z` | `MCPD_VERSION` | Install this release instead of the latest |
| `--user NAME` | `MCPD_USER` | Name of the first MCP user (default `mcp`; `""` to create none) |
| `--no-start` | | Install and enable the service, but don't start it (`sudo systemctl start mcpd` later) |
| `--archive FILE` | | Install from a downloaded release archive (offline hosts - see below) |
| `--uninstall` | | Stop and remove mcpd; keeps `/etc/mcpd` |
| `--uninstall --purge` | | Also delete `/etc/mcpd` |
| `--bin-dir DIR` | | macOS only: where to install `linuxctl` |

### Offline hosts

The release archive doesn't contain the installer. On a machine with internet access, download three files for your architecture (`amd64` or `arm64`) - the archive, `checksums.txt` and `install.sh`:

```bash
V=0.1.0 ARCH=amd64
curl -fsSLO https://github.com/nucleusv/linux-mcp-daemon/releases/download/v$V/linux-mcp-daemon_${V}_linux_${ARCH}.tar.gz
curl -fsSLO https://github.com/nucleusv/linux-mcp-daemon/releases/download/v$V/checksums.txt
curl -fsSLO https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh
```

Copy all three into one directory on the offline host and run:

```bash
sudo bash install.sh --archive linux-mcp-daemon_0.1.0_linux_amd64.tar.gz
```

With `checksums.txt` next to the archive the script verifies its sha256 (and refuses a mismatch); without it, it warns and installs unverified.

### Upgrading

Run the same command again. Binaries are replaced (the service is restarted only if `mcpd` actually changed); **existing configs in `/etc/mcpd/configs` are never overwritten**, and no extra user is created. Without `--version` you get the latest release; with `--version` you get exactly that one.

**Upgrading from v0.1.0.** Existing tokens keep working, and two things differ from a fresh install:

- Users stay in `daemon.yaml` (mcpd logs a warning) until the next `linuxctl create|update|delete mcpd user` moves them to `users.yaml` - nothing to do by hand.
- No user may apply config changes (`daemon/reload-config`) yet, so new users would only work after a restart. The installer says so; grant it once to your first user and restart mcpd this one time:

  ```bash
  sudo -E /usr/local/bin/linuxctl edit mcpd config sudo --config-path /etc/mcpd/configs
  ```

  It opens the file in `$VISUAL` or `$EDITOR` (default `vi`) - `sudo -E` passes yours through; plain `sudo` drops it. Under your user (`mcp`), replace `tools: {}` with

  ```yaml
        tools:
          daemon/reload-config:
            allowed: true
  ```

  save, and run `sudo systemctl restart mcpd`. (On save `linuxctl` tries to apply the change and reports `Not applied: user mcp is not authorized to run daemon/reload-config` - expected, the restart applies it.) From then on, [adding users](#adding-more-users) needs no restart.

### Adding more users

Each MCP user needs an OS account of the same name - every tool call runs as that account. For a user `alice`:

```bash
export MCP_SERVER=http://127.0.0.1:9091 MCP_TOKEN=<the mcp user's token>
sudo useradd --system --shell /usr/sbin/nologin alice
sudo -E /usr/local/bin/linuxctl create mcpd user alice --config-path /etc/mcpd/configs   # prints alice's token once
```

The output ends with the reload:

```text
Asking mcpd at http://127.0.0.1:9091 to reload its config...
Reloaded configs/daemon.yaml, configs/users.yaml and configs/mcp-sudo.yaml.
Changes:
  user alice: added
```

`linuxctl` writes the files and then asks the running mcpd to reload them (as the `mcp` user - that's what `MCP_TOKEN` and `sudo -E` are for), so `alice` can connect right away - no restart. (Its "Next steps" repeat the `useradd` you already ran. The full path matters on RHEL, Rocky, Alma and Fedora, whose `sudo` doesn't search `/usr/local/bin`: `sudo linuxctl` fails there with `command not found`.) If it prints `Not applied: user mcp is not authorized to run daemon/reload-config` instead, your install predates that tool - see [Upgrading from v0.1.0](#upgrading).

`alice` can now call every tool as her own OS account. To let her run specific tools as root, grant them in `mcp-sudo.yaml` - `sudo -E /usr/local/bin/linuxctl edit mcpd config sudo --config-path /etc/mcpd/configs` opens it in your editor, checks it when you save (like `visudo`) and applies it; see [mcp-sudo.yaml](./configuration/mcp-sudo). More commands (list, rotate, delete): [Daemon User Administration](./linuxctl/mcpd-admin).

### Uninstalling

```bash
curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | sudo bash -s -- --uninstall           # keeps /etc/mcpd
curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | sudo bash -s -- --uninstall --purge   # also deletes /etc/mcpd
```

This removes the service, the binaries and the man pages. Reinstalling after a plain `--uninstall` reuses the kept configs, so the existing users and tokens keep working (and no new token is printed). With `--archive` (offline hosts), uninstall with the same local copy: `sudo bash install.sh --uninstall`. **The OS accounts** created for MCP users (`mcp`, plus any you added) are left in place either way - remove them with `sudo userdel -r mcp`.

### Optional system tools

A few tools wrap standard binaries - `disks/health` (`smartctl`), `network/trace-path` (`traceroute`), `files/filetype` (`file`), `network/connections` (`ss`), `logs/journal-control` (`journalctl`). The installer warns about missing ones; on Debian/Ubuntu:

```bash
sudo apt install smartmontools traceroute file iproute2
```

On RHEL/Fedora: `sudo dnf install smartmontools traceroute file iproute`.

## Container image

On a Linux host with Docker (run as root, or drop `sudo` if your user is in the `docker` group):

```bash
IMAGE=ghcr.io/nucleusv/linux-mcp-daemon:latest   # or a fixed version, e.g. :0.1.0

# 1. Seed a configs directory on the host from the image's clean defaults
sudo mkdir -p /etc/mcpd/configs
sudo docker run --rm -v /etc/mcpd/configs:/out --entrypoint cp $IMAGE -r /root/configs/. /out/
sudo chmod 600 /etc/mcpd/configs/*.yaml    # they will hold token hashes

# 2. Create a user and print its token (once) - save it. It may also apply
#    config changes later (daemon/reload-config); mcpd isn't running yet.
sudo docker run --rm -v /etc/mcpd/configs:/root/configs $IMAGE \
  linuxctl create mcpd user privileged --grant daemon/reload-config --no-reload --config-path /root/configs

# 3. Run, administering the real host
sudo docker run -d --name mcpd --restart unless-stopped \
  --network host --pid host --privileged \
  --mount type=bind,source=/,target=/host,readonly,bind-propagation=rslave \
  -v /etc/mcpd/configs:/root/configs \
  $IMAGE
```

Step 2's "Next steps" text is written for the systemd install - in the container ignore its `useradd` line (the account already exists in the image, and step 3 starts mcpd with the new user).

Try it - the image contains `linuxctl`, so you don't need it on the host:

```bash
sudo docker exec -e MCP_TOKEN=<token from step 2> mcpd linuxctl get system os-release
```

or from any machine with `linuxctl` installed: `export MCP_SERVER=http://<host>:9091 MCP_TOKEN=<token>`, then `linuxctl get system os-release`.

- **MCP users must be OS accounts that exist inside the image** - `testuser`, `unpriviliged` (spelled that way) and `privileged` are built in; the name decides which UID each call runs as. For other names, build your own image `FROM` this one with `RUN useradd -m NAME`.
- **Unprivileged calls see the container, not the host.** A call reaches the real host only when it is made with `privileged: true` (`--privileged true` in `linuxctl`) *and* the user is granted that tool in `mcp-sudo.yaml` - the image's config sets `worker.containerized: true`, so such calls join the host's mount namespace. `--network host --pid host --privileged` plus the read-only host root mount make that possible; without them, mcpd only administers its own container. The user created in step 2 is granted only `daemon/reload-config` - nothing as root (its name, `privileged`, is just the OS account). To grant it, e.g., `system/os-release`, replace the contents of `/etc/mcpd/configs/mcp-sudo.yaml` with (e.g. `sudo nano /etc/mcpd/configs/mcp-sudo.yaml`):

  ```yaml
  users:
    privileged:
      privileged:
        tools:
          daemon/reload-config:
            allowed: true
          system/os-release:
            allowed: true
        resources: {}
  ```

  then apply it with `sudo docker exec -e MCP_TOKEN=<token> mcpd linuxctl reload daemon` and call `linuxctl get system os-release --privileged true` - it now reports the host's OS. If the file has a mistake, the reload says so and mcpd keeps the config it has. All options: [mcp-sudo.yaml](./configuration/mcp-sudo).
- The image ships **no users and no tokens** - step 2 is required. With mcpd running, add more users inside the container, which also applies the change: `sudo docker exec -e MCP_TOKEN=<token> mcpd linuxctl create mcpd user testuser --config-path /root/configs`. After changing files any other way (a `docker run --rm ... linuxctl ...` as in step 2, or an editor), apply them with `sudo docker exec -e MCP_TOKEN=<token> mcpd linuxctl reload daemon` - or `sudo docker restart mcpd`.

## macOS: the CLI

`linuxctl` runs anywhere and talks to a remote mcpd. The same install script, run on a Mac, installs just `linuxctl` (from the macOS release archive, sha256-verified) - no sudo needed when the target directory is yours:

```bash
curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | bash -s -- --bin-dir ~/.local/bin
export PATH="$HOME/.local/bin:$PATH"     # ~/.local/bin isn't on macOS's default PATH
export MCP_SERVER=http://your-server:9091 MCP_TOKEN=<token>
linuxctl get system os-release
```

The `export PATH` line lasts for the current terminal; add it to `~/.zshrc` to keep it. Without `--bin-dir` the script installs to `/usr/local/bin` (run it with `sudo bash` instead of `bash` if that isn't writable). To remove it: `curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | bash -s -- --uninstall --bin-dir ~/.local/bin`. Or download `linuxctl_<version>_darwin_arm64.tar.gz` (Apple Silicon) / `..._darwin_amd64.tar.gz` (Intel) from the [release page](https://github.com/nucleusv/linux-mcp-daemon/releases) yourself.

Shell completion: see [Autocompletion](./linuxctl/autocompletion).

## Before exposing mcpd

- mcpd listens on **all interfaces**, and speaks **plain HTTP** unless TLS is enabled in `daemon.yaml` (`server.tls`). Bearer tokens travel in every request - firewall the port to trusted addresses, or enable TLS, before reaching it over a network.
- Tokens only authenticate; what a user may do **as root** is granted per tool in `mcp-sudo.yaml` - see [mcp-sudo.yaml](./configuration/mcp-sudo). Without grants, a user can call every tool, but only as its own OS account.
- To add more users, see [Adding more users](#adding-more-users) above.

## Connecting an AI agent

See [AI Agent Configuration](./ai-agent-configuration) for Claude Code, Claude Desktop and other MCP clients.
