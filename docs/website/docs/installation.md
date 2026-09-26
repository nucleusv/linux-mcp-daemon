---
sidebar_position: 2.5
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
==> Installed. mcpd listens on https port 9091 on all interfaces.

  MCP user:  mcp
  Token:     4c1f...e9a2
             (shown once - only a salted hash is stored; save it now)

  Try it:
    export MCP_SERVER=https://127.0.0.1:9091
    export MCP_TLS_FINGERPRINT=sha256:4527...0175   # trusts mcpd's self-signed certificate
    export MCP_TOKEN=4c1f...e9a2
    linuxctl get system os-release
```

Run those commands (with your token and fingerprint) as any user on the host - `linuxctl` talks to mcpd over HTTPS, so no `sudo` is needed. mcpd serves **TLS by default**, with a self-signed certificate it created on first start in `/etc/mcpd/configs/tls/`; `MCP_TLS_FINGERPRINT` pins it (`linuxctl describe mcpd tls` shows it again; as root on the host, `linuxctl` trusts the file directly). Plain HTTP is off (`server.http` in `daemon.yaml`) - see [Daemon configuration](./configuration/daemon). `linuxctl get processes top` is another quick check.

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
export MCP_SERVER=https://127.0.0.1:9091 MCP_TLS_FINGERPRINT=<fingerprint> MCP_TOKEN=<the mcp user's token>
sudo useradd --system --shell /usr/sbin/nologin alice
sudo -E /usr/local/bin/linuxctl create mcpd user alice --config-path /etc/mcpd/configs   # prints alice's token once
```

The output ends with the reload:

```text
Asking mcpd at https://127.0.0.1:9091 to reload its config...
Reloaded configs/daemon.yaml, configs/users.yaml and configs/mcp-sudo.yaml.
Changes:
  user alice: added
```

`linuxctl` writes the files and then asks the running mcpd to reload them (as the `mcp` user - that's what `MCP_TOKEN` and `sudo -E` are for), so `alice` can connect right away - no restart. (Its "Next steps" repeat the `useradd` you already ran. The full path matters on RHEL, Rocky, Alma and Fedora, whose `sudo` doesn't search `/usr/local/bin`: `sudo linuxctl` fails there with `command not found`.) If it prints `Not applied: user mcp is not authorized to run daemon/reload-config` instead, your install predates that tool - see [Upgrading from v0.1.0](#upgrading).

`alice` can now call every tool as her own OS account. To let her run specific tools as root, grant them in `mcp-sudo.yaml` - `sudo -E /usr/local/bin/linuxctl edit mcpd config sudo --config-path /etc/mcpd/configs` opens it in your editor, checks it when you save (like `visudo`) and applies it; see [mcp-sudo.yaml](./configuration/mcp-sudo), and before granting anything as root, [Permissions and Risks](./configuration/permissions-and-risks) - some grants amount to full root. More commands (list, rotate, delete): [Daemon User Administration](./linuxctl/mcpd-admin).

### Uninstalling

```bash
curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | sudo bash -s -- --uninstall           # keeps /etc/mcpd
curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | sudo bash -s -- --uninstall --purge   # also deletes /etc/mcpd
```

This removes the service, the binaries and the man pages. Reinstalling after a plain `--uninstall` reuses the kept configs, so the existing users and tokens keep working (and no new token is printed). With `--archive` (offline hosts), uninstall with the same local copy: `sudo bash install.sh --uninstall`. **The OS accounts** created for MCP users (`mcp`, plus any you added) are left in place either way - remove them with `sudo userdel -r mcp`.

### Optional system tools

A few tools wrap standard binaries - `disks/health` (`smartctl`), `network/trace-path` (`traceroute`), `logs/journal-control` (`journalctl`, part of systemd). Everything else reads the kernel directly. The installer warns about missing ones; on Debian/Ubuntu:

```bash
sudo apt install smartmontools traceroute
```

On RHEL/Fedora: `sudo dnf install smartmontools traceroute`.

## Packages (.deb, .rpm)

Every release also ships packages for amd64 and arm64: the same `mcpd` and `linuxctl`, installed to `/usr/bin` with a systemd unit and man pages. Use them instead of the script if you'd rather have the package manager own the install.

**Debian / Ubuntu:**

```bash
V=0.3.4 ARCH=amd64    # or arm64
curl -fsSLO https://github.com/nucleusv/linux-mcp-daemon/releases/download/v$V/linux-mcp-daemon_${V}_${ARCH}.deb
sudo apt install ./linux-mcp-daemon_${V}_${ARCH}.deb
```

**RHEL / Rocky / Alma / Fedora:**

```bash
V=0.3.4 ARCH=x86_64   # or aarch64
curl -fsSLO https://github.com/nucleusv/linux-mcp-daemon/releases/download/v$V/linux-mcp-daemon-${V}-1.${ARCH}.rpm
sudo dnf install ./linux-mcp-daemon-${V}-1.${ARCH}.rpm
```

To verify the download first, fetch the release's `checksums.txt` next to it and run `sha256sum --ignore-missing -c checksums.txt`.

The package pulls in `smartmontools` and `traceroute` as recommended packages, creates `/etc/mcpd/configs` (mode `0750`, files `0600`) from clean templates - **no users, no tokens** - and enables the `mcpd` service **without starting it**. It prints the next steps:

```text
mcpd is installed and enabled, but not started - it has no users yet:
  useradd --system --create-home --shell /usr/sbin/nologin mcp
  linuxctl create mcpd user mcp --grant daemon/reload-config --config-path /etc/mcpd/configs   # prints its token once
  systemctl start mcpd
```

Run them with `sudo`, save the token, then try it as in the script install: `export MCP_SERVER=https://127.0.0.1:9091 MCP_TLS_FINGERPRINT=<fingerprint> MCP_TOKEN=<token>` (the fingerprint: `sudo linuxctl describe mcpd tls --config-path /etc/mcpd/configs`, once mcpd has started and created its certificate) and `linuxctl get system os-release`. More users and grants work the same way too - see [Adding more users](#adding-more-users) (with `/usr/bin/linuxctl`, which `sudo` finds on every distribution).

- **Upgrade:** install the newer package the same way. Configs in `/etc/mcpd/configs` are never touched (they aren't package-managed conffiles - `linuxctl` rewrites them, so there are no upgrade prompts), and a running mcpd is restarted on the new binary.
- **Remove:** `sudo apt remove linux-mcp-daemon` / `sudo dnf remove linux-mcp-daemon` stops mcpd and keeps `/etc/mcpd`; reinstalling picks the users up again. `sudo apt purge linux-mcp-daemon` also deletes `/etc/mcpd` (with rpm, delete it by hand). MCP users' OS accounts stay either way.
- **Switching from the script:** run the script's `--uninstall` first (it keeps `/etc/mcpd/configs`, which the package then uses as they are - existing users and tokens keep working). Otherwise the script's `/etc/systemd/system/mcpd.service` overrides the package's unit and keeps running `/usr/local/bin/mcpd`.
- **Without systemd** (e.g. in a container) the package installs the same and prints how to start mcpd by hand: `cd /etc/mcpd && mcpd --config-dir /etc/mcpd/configs`.

## Container image

On a Linux host with Docker (run as root, or drop `sudo` if your user is in the `docker` group):

```bash
IMAGE=ghcr.io/nucleusv/linux-mcp-daemon:latest   # or a fixed version, e.g. :0.3.4

# 1. Seed a configs directory on the host from the image's clean defaults
sudo mkdir -p /etc/mcpd/configs
sudo docker run --rm -v /etc/mcpd/configs:/out --entrypoint cp $IMAGE -r /etc/mcpd/configs/. /out/
sudo chmod 600 /etc/mcpd/configs/*.yaml    # they will hold token hashes

# 2. Create a user and print its token (once) - save it. It may also apply
#    config changes later (daemon/reload-config); mcpd isn't running yet.
sudo docker run --rm -v /etc/mcpd/configs:/etc/mcpd/configs $IMAGE \
  linuxctl create mcpd user privileged --grant daemon/reload-config --no-reload

# 3. Run, administering the real host
sudo docker run -d --name mcpd --restart unless-stopped \
  --network host --pid host --privileged \
  --mount type=bind,source=/,target=/host,readonly,bind-propagation=rslave \
  -v /etc/mcpd/configs:/etc/mcpd/configs \
  $IMAGE
```

The configs sit at the same path inside the container as on the host, `/etc/mcpd/configs` (the image sets `MCPD_CONFIG_DIR` to it, so `linuxctl` inside needs no `--config-path`). Images up to 0.1.0 kept them in `/root/configs`; a `-v ...:/root/configs` mount still works with newer images.

Step 2's "Next steps" text is written for the systemd install - in the container ignore its `useradd` line (the account already exists in the image, and step 3 starts mcpd with the new user).

Try it - the image contains `linuxctl`, so you don't need it on the host:

```bash
sudo docker exec -e MCP_TOKEN=<token from step 2> mcpd linuxctl get system os-release
```

or from any machine with `linuxctl` installed: `export MCP_SERVER=https://<host>:9091 MCP_TLS_FINGERPRINT=<fingerprint> MCP_TOKEN=<token>`, then `linuxctl get system os-release`. mcpd creates its self-signed certificate in `/etc/mcpd/configs/tls/` on first start (the configs mount must be writable) and logs the fingerprint - `sudo docker logs mcpd`, or `sudo docker exec mcpd linuxctl describe mcpd tls`.

- **MCP users must be OS accounts that exist inside the image** - `testuser`, `unpriviliged` (spelled that way) and `privileged` are built in; the name decides which UID each call runs as. For other names, build your own image `FROM` this one with `RUN useradd -m NAME`.
- **Docker's `--privileged` is not a call's `privileged: true`.** The first lets the container reach the host; the second asks for root on one call, and only works for tools granted in `mcp-sudo.yaml` - see [the difference, and what happens without the Docker flags](./configuration/mcp-sudo#host-filesystem-access-containers).
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
- The image ships **no users and no tokens** - step 2 is required. With mcpd running, add more users inside the container, which also applies the change: `sudo docker exec -e MCP_TOKEN=<token> mcpd linuxctl create mcpd user testuser`. After changing files any other way (a `docker run --rm ... linuxctl ...` as in step 2, or an editor), apply them with `sudo docker exec -e MCP_TOKEN=<token> mcpd linuxctl reload daemon` - or `sudo docker restart mcpd`.

## macOS: the CLI

`linuxctl` runs anywhere and talks to a remote mcpd. The same install script, run on a Mac, installs just `linuxctl` (from the macOS release archive, sha256-verified) - no sudo needed when the target directory is yours:

```bash
curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | bash -s -- --bin-dir ~/.local/bin
export PATH="$HOME/.local/bin:$PATH"     # ~/.local/bin isn't on macOS's default PATH
export MCP_SERVER=https://your-server:9091 MCP_TLS_FINGERPRINT=<fingerprint> MCP_TOKEN=<token>   # fingerprint: linuxctl describe mcpd tls, on the server
linuxctl get system os-release
```

The `export PATH` line lasts for the current terminal; add it to `~/.zshrc` to keep it. Without `--bin-dir` the script installs to `/usr/local/bin` (run it with `sudo bash` instead of `bash` if that isn't writable). To remove it: `curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | bash -s -- --uninstall --bin-dir ~/.local/bin`. Or download `linuxctl_<version>_darwin_arm64.tar.gz` (Apple Silicon) / `..._darwin_amd64.tar.gz` (Intel) from the [release page](https://github.com/nucleusv/linux-mcp-daemon/releases) yourself.

Shell completion: see [Autocompletion](./linuxctl/autocompletion).

## Before exposing mcpd

- mcpd listens on **all interfaces**, over **TLS** by default (a self-signed certificate - clients pin its fingerprint or trust the file; or install a CA-issued one, see [Daemon configuration](./configuration/daemon)). Don't turn on plain HTTP (`server.http`) for anything but a trusted network or a TLS-terminating proxy: bearer tokens travel in every request. Firewalling the port to the addresses that need it is still a good idea.
- Tokens only authenticate; what a user may do **as root** is granted per tool in `mcp-sudo.yaml` - see [mcp-sudo.yaml](./configuration/mcp-sudo). Without grants, a user can call every tool, but only as its own OS account. Map each MCP user to a dedicated, unprivileged OS account (not root, not in `docker`/`lxd`/`disk`), and read [Permissions and Risks](./configuration/permissions-and-risks) before granting root.
- To add more users, see [Adding more users](#adding-more-users) above.

## Connecting an AI agent

See [AI Agent Configuration](./ai-agent-configuration) for Claude Code, Claude Desktop and other MCP clients.
