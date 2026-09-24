#!/bin/sh
# linux-mcp-daemon package: after install or upgrade (deb and rpm).
# deb: $1=configure, $2=previously installed version (empty on first install)
# rpm: $1=1 on first install, >=2 on upgrade
set -e

first=0
case "$1" in
    configure) [ -z "$2" ] && first=1 ;;
    1) first=1 ;;
esac

# Configs work like scripts/install.sh: templates are copied only where no
# config exists yet, so an upgrade never touches the admin's files (users
# and grants that `linuxctl` rewrites). No conffile prompts either.
conf=/etc/mcpd/configs
install -d -m 0750 "$conf"
for f in daemon.yaml users.yaml mcp-sudo.yaml; do
    if [ ! -e "$conf/$f" ]; then
        # Configs from before users.yaml keep their users in daemon.yaml;
        # an empty users.yaml next to them would stop mcpd from starting.
        if [ "$f" = users.yaml ] && [ "$first" -eq 0 ] && [ -e "$conf/daemon.yaml" ]; then
            continue
        fi
        install -m 0600 "/usr/share/linux-mcp-daemon/configs/$f" "$conf/$f"
    fi
done

if [ -d /run/systemd/system ]; then
    systemctl daemon-reload >/dev/null 2>&1 || true
    if [ "$first" -eq 1 ]; then
        systemctl enable mcpd >/dev/null 2>&1 || true
    else
        systemctl try-restart mcpd >/dev/null 2>&1 || true
    fi
fi

if [ "$first" -eq 1 ] && ! grep -q 'username' "$conf/users.yaml" 2>/dev/null; then
    if [ -d /run/systemd/system ]; then
        state="installed and enabled, but not started"
        start="systemctl start mcpd"
    else
        state="installed (no systemd: it won't start on its own)"
        start="cd /etc/mcpd && mcpd --config-dir $conf   # runs in the foreground"
    fi
    cat <<MSG

mcpd is $state - it has no users yet:
  useradd --system --create-home --shell /usr/sbin/nologin mcp
  linuxctl create mcpd user mcp --grant daemon/reload-config --config-path $conf   # prints its token once
  $start
Docs: https://nucleusv.github.io/linux-mcp-daemon/installation/#packages-deb-rpm

MSG
fi
exit 0
