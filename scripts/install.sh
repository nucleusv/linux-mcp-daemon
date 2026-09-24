#!/usr/bin/env bash
# Install, upgrade or uninstall linux-mcp-daemon (mcpd + linuxctl) on a
# systemd Linux host from a GitHub release. On macOS it installs only the
# linuxctl CLI (mcpd is Linux-only).
#
#   curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | sudo bash
#
# Options (after `bash -s --` when piping):
#   --version vX.Y.Z   install this release (default: the latest)
#   --user NAME        MCP user to create on first install (default: mcp);
#                      its token is printed once. --user "" skips this.
#   --no-start         install and enable, but don't start the service
#   --archive FILE     install from a local release tarball instead of
#                      downloading (offline installs, testing); verified
#                      if the release's checksums.txt sits next to it
#   --uninstall        stop and remove mcpd (keeps /etc/mcpd)
#   --purge            with --uninstall: also delete /etc/mcpd
#   --bin-dir DIR      macOS only: where to put linuxctl (default /usr/local/bin)
#
# Environment variables work too - put them after sudo, which drops
# variables exported in your shell:
#   curl -fsSL .../install.sh | sudo MCPD_VERSION=v0.1.0 MCPD_USER=name bash
#
# Re-running on an installed host upgrades the binaries (restarting the
# service only if they changed); existing configs in /etc/mcpd/configs are
# never overwritten.
#
# Everything runs inside main(), called on the last line: if a
# `curl | bash` download is cut off midway, bash never executes a
# half-received script.
set -euo pipefail

REPO="nucleusv/linux-mcp-daemon"
BIN_DIR="/usr/local/bin"
CONF_ROOT="/etc/mcpd"
CONF_DIR="$CONF_ROOT/configs"
UNIT="/etc/systemd/system/mcpd.service"
MAN_DIR="/usr/local/share/man"

VERSION="${MCPD_VERSION:-}"
MCP_USER="${MCPD_USER-mcp}"
START=1
ARCHIVE=""
UNINSTALL=0
PURGE=0
BIN_DIR_SET=0

say()  { printf '\033[1;32m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

# fetch URL OUT - curl, or wget where curl isn't installed.
fetch() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$2" "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$2" "$1"
    else
        die "curl or wget is required"
    fi
}

# sha256 FILE - sha256sum, or shasum where coreutils' tool is missing.
sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        die "sha256sum or shasum is required to verify the download"
    fi
}

usage() {
    cat <<'USAGE'
Install, upgrade or uninstall linux-mcp-daemon (mcpd + linuxctl).

  curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | sudo bash -s -- [options]

  --version vX.Y.Z   install this release (default: latest)      env: MCPD_VERSION
  --user NAME        MCP user created on first install (default: mcp; "" to skip)  env: MCPD_USER
  --no-start         install and enable, but don't start the service
  --archive FILE     install from a local release tarball (sha256-verified
                     when the release's checksums.txt is in the same directory)
  --uninstall        remove mcpd (keeps /etc/mcpd); add --purge to delete it too
  --bin-dir DIR      macOS only: where to put linuxctl (default /usr/local/bin)

On macOS only the linuxctl CLI is installed - no sudo needed with a writable --bin-dir:
  curl -fsSL https://raw.githubusercontent.com/nucleusv/linux-mcp-daemon/main/scripts/install.sh | bash -s -- --bin-dir ~/.local/bin
USAGE
}

# resolve_version sets VERSION (latest release if unset) and NUM.
resolve_version() {
    if [ -z "$VERSION" ]; then
        fetch "https://api.github.com/repos/$REPO/releases/latest" "$TMP/latest.json" \
            || die "could not query the latest release (try --version vX.Y.Z)"
        VERSION="$(sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' "$TMP/latest.json" | head -n1)"
        [ -n "$VERSION" ] || die "could not determine the latest release (try --version vX.Y.Z)"
    fi
    case "$VERSION" in v*) ;; *) VERSION="v$VERSION" ;; esac
    NUM="${VERSION#v}"
}

# download_verified FILE - fetch a release asset into $TMP/FILE and check
# it against the release's checksums.txt, aborting on any mismatch.
download_verified() {
    local base="https://github.com/$REPO/releases/download/$VERSION"
    say "Downloading $1 ($VERSION)"
    fetch "$base/$1" "$TMP/$1" || die "download failed: $base/$1"
    fetch "$base/checksums.txt" "$TMP/checksums.txt" || die "download failed: $base/checksums.txt"
    say "Verifying sha256 checksum"
    local expected actual
    expected="$(awk -v f="$1" '$2 == f {print $1}' "$TMP/checksums.txt")"
    [ -n "$expected" ] || die "$1 is not listed in checksums.txt"
    actual="$(sha256 "$TMP/$1")"
    [ "$expected" = "$actual" ] || die "checksum mismatch for $1 (expected $expected, got $actual) - refusing to install"
}

# install_cli_macos installs just linuxctl: mcpd is Linux-only, and a Mac
# typically drives a remote mcpd. No root needed if --bin-dir is writable.
install_cli_macos() {
    [ -z "$ARCHIVE" ] || die "--archive is for Linux server archives"
    [ "$UNINSTALL" -eq 0 ] || { rm -f "$BIN_DIR/linuxctl"; say "linuxctl removed from $BIN_DIR"; return; }
    case "$(uname -m)" in
        arm64)  ARCH="arm64" ;;
        x86_64) ARCH="amd64" ;;
        *) die "unsupported architecture: $(uname -m)" ;;
    esac
    mkdir -p "$BIN_DIR" 2>/dev/null || true
    [ -w "$BIN_DIR" ] || die "$BIN_DIR is not writable - rerun with sudo, or pass --bin-dir ~/.local/bin"
    TMP="$(mktemp -d)"
    trap 'rm -rf "$TMP"' EXIT
    resolve_version
    local file="linuxctl_${NUM}_darwin_${ARCH}.tar.gz"
    download_verified "$file"
    tar -xzf "$TMP/$file" -C "$TMP" linuxctl
    install -m 0755 "$TMP/linuxctl" "$BIN_DIR/linuxctl"
    say "Installed $("$BIN_DIR/linuxctl" --version) to $BIN_DIR"
    case ":$PATH:" in *":$BIN_DIR:"*) ;; *) warn "$BIN_DIR is not on your PATH - add: export PATH=\"$BIN_DIR:\$PATH\"" ;; esac
    cat <<EOF

  Point it at a Linux host running mcpd:
    export MCP_SERVER=http://your-server:9091
    export MCP_TOKEN=<token>
    linuxctl get system os-release

  Shell completion: source <(linuxctl completion zsh)   # or bash
EOF
}

main() {
while [ $# -gt 0 ]; do
    case "$1" in
        --version)   VERSION="${2:?--version needs a value}"; shift 2 ;;
        --user)      MCP_USER="${2-}"; shift 2 ;;
        --no-start)  START=0; shift ;;
        --archive)   ARCHIVE="${2:?--archive needs a file}"; shift 2 ;;
        --uninstall) UNINSTALL=1; shift ;;
        --purge)     PURGE=1; shift ;;
        --bin-dir)   BIN_DIR="${2:?--bin-dir needs a directory}"; BIN_DIR_SET=1; shift 2 ;;
        -h|--help)   usage; exit 0 ;;
        *)           die "unknown option: $1 (see --help)" ;;
    esac
done

case "$(uname -s)" in
    Darwin) install_cli_macos; exit 0 ;;
    Linux)  ;;
    *) die "unsupported OS: $(uname -s) (mcpd runs on Linux; linuxctl also on macOS)" ;;
esac
[ "$BIN_DIR_SET" -eq 0 ] || die "--bin-dir is for macOS; on Linux the service expects $BIN_DIR"
[ "$(id -u)" -eq 0 ] || die "run as root (sudo bash)"
HAVE_SYSTEMD=0
if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
    HAVE_SYSTEMD=1
fi

# ---------------------------------------------------------------- uninstall
if [ "$UNINSTALL" -eq 1 ]; then
    if [ "$HAVE_SYSTEMD" -eq 1 ] && [ -f "$UNIT" ]; then
        systemctl disable --now mcpd 2>/dev/null || true
    fi
    rm -f "$UNIT" "$BIN_DIR/mcpd" "$BIN_DIR/linuxctl" "$MAN_DIR/man8/mcpd.8" "$MAN_DIR/man1/linuxctl.1"
    [ "$HAVE_SYSTEMD" -eq 1 ] && systemctl daemon-reload
    if [ "$PURGE" -eq 1 ]; then
        rm -rf "$CONF_ROOT"
        say "mcpd uninstalled, $CONF_ROOT deleted"
    else
        say "mcpd uninstalled ($CONF_ROOT kept - use --purge to delete it)"
    fi
    echo "  OS accounts created for MCP users (e.g. mcp) are kept; remove one with: userdel -r NAME"
    exit 0
fi

# ------------------------------------------------------------------ fetch
case "$(uname -m)" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) die "unsupported architecture: $(uname -m) (releases cover amd64 and arm64)" ;;
esac

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

if [ -n "$ARCHIVE" ]; then
    [ -f "$ARCHIVE" ] || die "archive not found: $ARCHIVE"
    # Verify against the release's checksums.txt when it sits next to the
    # archive (the offline instructions say to download both).
    sums="$(dirname "$ARCHIVE")/checksums.txt"
    name="$(basename "$ARCHIVE")"
    if [ -f "$sums" ]; then
        expected="$(awk -v f="$name" '$2 == f {print $1}' "$sums")"
        [ -n "$expected" ] || die "$name is not listed in $sums"
        [ "$expected" = "$(sha256 "$ARCHIVE")" ] || die "checksum mismatch for $name - refusing to install"
        say "Installing from local archive $ARCHIVE (sha256 verified against $sums)"
    else
        warn "no checksums.txt next to $ARCHIVE - installing it unverified"
    fi
    cp "$ARCHIVE" "$TMP/release.tar.gz"
else
    resolve_version
    FILE="linux-mcp-daemon_${NUM}_linux_${ARCH}.tar.gz"
    download_verified "$FILE"
    mv "$TMP/$FILE" "$TMP/release.tar.gz"
fi

mkdir -p "$TMP/x"
tar -xzf "$TMP/release.tar.gz" -C "$TMP/x"
for f in mcpd linuxctl packaging/mcpd.service packaging/configs/daemon.yaml packaging/configs/mcp-sudo.yaml; do
    [ -e "$TMP/x/$f" ] || die "release archive is missing $f"
done

# ---------------------------------------------------------------- install
UPGRADE=0
CHANGED=1
if [ -x "$BIN_DIR/mcpd" ]; then
    UPGRADE=1
    # Like k3s: an unchanged binary means no restart (and no dropped
    # client sessions) when re-running the installer.
    if [ "$(sha256 "$BIN_DIR/mcpd")" = "$(sha256 "$TMP/x/mcpd")" ]; then
        CHANGED=0
    fi
fi

say "Installing mcpd and linuxctl to $BIN_DIR"
install -m 0755 "$TMP/x/mcpd" "$BIN_DIR/mcpd.new"
install -m 0755 "$TMP/x/linuxctl" "$BIN_DIR/linuxctl.new"
mv -f "$BIN_DIR/mcpd.new" "$BIN_DIR/mcpd"          # atomic replace, even while running
mv -f "$BIN_DIR/linuxctl.new" "$BIN_DIR/linuxctl"

if [ -d "$TMP/x/docs/man" ]; then
    install -D -m 0644 "$TMP/x/docs/man/mcpd.8" "$MAN_DIR/man8/mcpd.8"
    install -D -m 0644 "$TMP/x/docs/man/linuxctl.1" "$MAN_DIR/man1/linuxctl.1"
fi

install -d -m 0750 "$CONF_DIR"
FRESH=0
UPGRADE=0
[ -e "$CONF_DIR/daemon.yaml" ] && UPGRADE=1
for f in daemon.yaml users.yaml mcp-sudo.yaml; do
    if [ -e "$CONF_DIR/$f" ]; then
        say "Keeping existing $CONF_DIR/$f"
    elif [ "$f" = users.yaml ] && { [ "$UPGRADE" -eq 1 ] || [ ! -e "$TMP/x/packaging/configs/$f" ]; }; then
        # Configs from before users.yaml keep their users in daemon.yaml;
        # an empty users.yaml next to them would make mcpd refuse to start
        # (users in two places). linuxctl moves them on its next user change.
        # Releases before users.yaml don't ship the file at all.
        :
    else
        install -m 0600 "$TMP/x/packaging/configs/$f" "$CONF_DIR/$f"
        [ "$f" = users.yaml ] || FRESH=1
    fi
done

# First install only: one MCP user, backed by a matching OS account (each
# tool call runs as that account). Its token is shown exactly once.
TOKEN_MSG=""
if [ "$FRESH" -eq 1 ] && [ -n "$MCP_USER" ]; then
    if ! id "$MCP_USER" >/dev/null 2>&1; then
        say "Creating OS account $MCP_USER (no login shell)"
        useradd --system --create-home --shell /usr/sbin/nologin "$MCP_USER"
    fi
    # The first user may also apply config changes (daemon/reload-config),
    # so users added later take effect without a restart. mcpd isn't
    # running yet, so there is nothing to reload now.
    OUT="$("$BIN_DIR/linuxctl" create mcpd user "$MCP_USER" --grant daemon/reload-config --no-reload --config-path "$CONF_DIR")"
    TOKEN="$(printf '%s\n' "$OUT" | awk '/Token/ {getline; gsub(/ /, ""); print; exit}')"
    [ -n "$TOKEN" ] || die "failed to create MCP user $MCP_USER: $OUT"
    TOKEN_MSG="$TOKEN"
fi

RUNNING=0
if [ "$HAVE_SYSTEMD" -eq 1 ]; then
    install -m 0644 "$TMP/x/packaging/mcpd.service" "$UNIT"
    systemctl daemon-reload
    systemctl enable mcpd >/dev/null 2>&1
    if [ "$START" -eq 1 ] && [ "$CHANGED" -eq 0 ] && [ -z "$TOKEN_MSG" ] && systemctl is-active --quiet mcpd; then
        say "mcpd binary unchanged - not restarting ($("$BIN_DIR/mcpd" --version))"
        RUNNING=1
    elif [ "$START" -eq 1 ]; then
        systemctl restart mcpd
        sleep 1
        systemctl is-active --quiet mcpd || die "mcpd failed to start - see: journalctl -u mcpd -n 50"
        say "mcpd is running ($("$BIN_DIR/mcpd" --version))"
        RUNNING=1
    else
        say "mcpd installed and enabled, not started (--no-start)"
    fi
else
    warn "systemd not detected - the service was not installed. Start it manually: cd $CONF_ROOT && $BIN_DIR/mcpd"
fi

# ------------------------------------------------------------------ report
missing=""
for bin in smartctl traceroute file ss journalctl; do
    command -v "$bin" >/dev/null 2>&1 || missing="$missing $bin"
done
[ -n "$missing" ] && warn "optional tools not found:$missing - the tools that wrap them won't work (Debian/Ubuntu: apt install smartmontools traceroute file iproute2)"

PORT="$(awk '/^server:/ {s=1} s && /^  port:/ {print $2; exit}' "$CONF_DIR/daemon.yaml")"
PORT="${PORT:-9091}"
echo
if [ "$UPGRADE" -eq 1 ]; then
    say "Upgrade complete. Configs in $CONF_DIR were left untouched."
elif [ "$RUNNING" -eq 1 ]; then
    say "Installed. mcpd listens on port $PORT on all interfaces."
elif [ "$HAVE_SYSTEMD" -eq 1 ]; then
    say "Installed, not started. Start it with: systemctl start mcpd (it will listen on port $PORT on all interfaces)"
else
    say "Installed, not started (no systemd). Start it with: cd $CONF_ROOT && $BIN_DIR/mcpd (it will listen on port $PORT on all interfaces)"
fi
if [ "$UPGRADE" -eq 0 ] && [ "$FRESH" -eq 0 ]; then
    echo "  Existing configs in $CONF_DIR were kept - their users and tokens still work."
fi
if [ "$FRESH" -eq 1 ] && [ -z "$MCP_USER" ]; then
    cat <<EOF

  No MCP user was created (--user ""). Add one - each needs a matching OS account:
    useradd --system --shell /usr/sbin/nologin NAME
    $BIN_DIR/linuxctl create mcpd user NAME --config-path $CONF_DIR
    systemctl restart mcpd
EOF
fi
if [ -n "$TOKEN_MSG" ]; then
    cat <<EOF

  MCP user:  $MCP_USER
  Token:     $TOKEN_MSG
             (shown once - only a salted hash is stored; save it now)

  Try it:
    export MCP_SERVER=http://127.0.0.1:$PORT
    export MCP_TOKEN=$TOKEN_MSG
    linuxctl get system os-release

EOF
fi
cat <<EOF
  Security: mcpd speaks plain HTTP unless TLS is enabled in $CONF_DIR/daemon.yaml.
  Firewall port $PORT to trusted addresses, or enable TLS, before exposing it.
  Root access for tools is granted per user in $CONF_DIR/mcp-sudo.yaml.
  Docs: https://nucleusv.github.io/linux-mcp-daemon/
EOF
}

main "$@"
