#!/bin/sh
# linux-mcp-daemon package: before removal (deb: $1=remove; rpm: $1=0).
# On upgrade (deb: upgrade, rpm: 1) the service keeps running - postinstall
# restarts it on the new binary.
case "$1" in
    remove|0)
        if [ -d /run/systemd/system ]; then
            systemctl disable --now mcpd >/dev/null 2>&1 || true
        fi
        ;;
esac
exit 0
