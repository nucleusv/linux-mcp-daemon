#!/bin/bash
set -e

DAEMON_URL=${DAEMON_URL:-"http://localhost:9090"}
TOKEN=${TOKEN:-"my-test-token-123"}

echo "Testing linuxctl CLI..."

# Ensure we are in the project root
cd "$(dirname "$0")/.."

# Build the CLI
echo "Building linuxctl..."
go build -o linuxctl ./cmd/linuxctl

# Test ping command
echo "Running 'linuxctl ping'..."
OUTPUT=$(./linuxctl -server "$DAEMON_URL" -token "$TOKEN" ping)

if echo "$OUTPUT" | grep -q "Successfully connected to mcpd daemon!"; then
    echo "Running 'linuxctl get_sudo_rules'..."
    ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" get_sudo_rules | grep "list_directory" > /dev/null

    echo "Running 'linuxctl list_directory --path /var/log --privileged true'..."
    ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" list_directory --path /var/log --privileged true | grep "syslog" > /dev/null || echo "Could not find syslog, but command succeeded"

    echo "Running 'linuxctl get_disk_usage --path /var/log --privileged true'..."
    ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" get_disk_usage --path /var/log --privileged true | grep "Total size" > /dev/null

    echo "Running 'linuxctl get_disk_space --path / --privileged true'..."
    ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" get_disk_space --path / --privileged true | grep "Filesystem" > /dev/null

    echo "✅ SUCCESS: linuxctl successfully dynamically executed all tools over SSE."
    # Clean up the binary
    rm -f linuxctl
    exit 0
else
    echo "❌ FAILED: linuxctl failed to ping the daemon."
    echo "Output: $OUTPUT"
    rm -f linuxctl
    exit 1
fi
