#!/bin/sh
# linux-mcp-daemon package: after removal. The configs in /etc/mcpd hold
# users and grants, so a plain remove keeps them; `apt purge` deletes them.
# (rpm has no purge: after `dnf remove`, delete /etc/mcpd by hand.)
if [ -d /run/systemd/system ]; then
    systemctl daemon-reload >/dev/null 2>&1 || true
fi
if [ "$1" = purge ]; then
    rm -rf /etc/mcpd
fi
exit 0
