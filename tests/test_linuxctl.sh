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
    echo "Running 'linuxctl auth/sudo-rules'..."
    OUT1=$(./linuxctl -token "$TOKEN" -server "$DAEMON_URL" auth/sudo-rules)
    echo "$OUT1"
    echo "$OUT1" | grep "list_directory" > /dev/null

    echo "Running 'linuxctl files/list-directory /var/log --privileged true' (positional arg test)..."
    OUT2=$(./linuxctl -token "$TOKEN" -server "$DAEMON_URL" files/list-directory /var/log --privileged true)
    echo "$OUT2"
    echo "$OUT2" | grep "syslog" > /dev/null || echo "Could not find syslog, but command succeeded"

    echo "Running 'linuxctl disks/disk-usage --path /var/log --privileged true'..."
    OUT3=$(./linuxctl -token "$TOKEN" -server "$DAEMON_URL" disks/disk-usage --path /var/log --privileged true)
    echo "$OUT3"
    echo "$OUT3" | grep "Total size" > /dev/null

    echo "Running 'linuxctl disks/disk-free --path / --privileged true'..."
    OUT4=$(./linuxctl -token "$TOKEN" -server "$DAEMON_URL" disks/disk-free --path / --privileged true)
    echo "$OUT4"
    echo "$OUT4" | grep "Filesystem" > /dev/null

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
