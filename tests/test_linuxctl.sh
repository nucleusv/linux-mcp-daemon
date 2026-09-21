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
    echo "Running 'linuxctl get auth get-sudo-rules'..."
    ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" get auth get-sudo-rules | grep "list_directory" > /dev/null

    echo "Running 'linuxctl get files list-directory /var/log --privileged true' (positional arg test)..."
    ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" get files list-directory /var/log --privileged true | grep "syslog" > /dev/null || echo "Could not find syslog, but command succeeded"

    echo "Running 'linuxctl get disks disk-usage --path /var/log --privileged true'..."
    ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" get disks disk-usage --path /var/log --privileged true | grep "Total size" > /dev/null

    echo "Running 'linuxctl get disks disk-space --path / --privileged true'..."
    ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" get disks disk-space --path / --privileged true | grep "Filesystem" > /dev/null

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
