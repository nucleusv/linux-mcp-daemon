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

if ! echo "$OUTPUT" | grep -q "Successfully connected to mcpd daemon!"; then
    echo "❌ FAILED: linuxctl failed to ping the daemon."
    echo "Output: $OUTPUT"
    rm -f linuxctl
    exit 1
fi

echo "==========================================="
echo "   Testing Tools (JSON & Table outputs)    "
echo "==========================================="

run_tool() {
    local cmd="$1"
    local name="$2"
    
    echo "Testing Tool: $name (Table Output)"
    ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" $cmd --output table
    
    echo "Testing Tool: $name (Wide Output)"
    ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" $cmd --output wide
    
    echo "Testing Tool: $name (YAML Output)"
    ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" $cmd --output yaml
    
    echo "Testing Tool: $name (JSON Output)"
    ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" $cmd --output json
}

run_tool "auth sudo-rules" "auth/sudo-rules"
run_tool "files list --path /var/log --privileged true" "files/list"
run_tool "disks usage --path /var/log --privileged true" "disks/usage"
run_tool "disks free --path / --privileged true" "disks/free"
run_tool "disks list" "disks/list"
run_tool "processes list --limit 5" "processes/list"
run_tool "network connections" "network/connections"
run_tool "network nslookup --host google.com" "network/nslookup"
run_tool "network curl --url http://127.0.0.1:9090/ping" "network/curl"
run_tool "network arp" "network/arp"
run_tool "network ping --host 127.0.0.1" "network/ping"
run_tool "memory usage" "memory/usage"
run_tool "cpu list" "cpu/list"
run_tool "cpu load-average" "cpu/load-average"

echo "==========================================="
echo "  Testing Resources (JSON & Table outputs) "
echo "==========================================="

run_resource() {
    local uri="$1"
    
    echo "Testing Resource: $uri (Table Output)"
    if [ "$uri" = "devices://dmi" ]; then
        ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" resource $uri --output table || echo "(Expected failure on some VMs)"
    else
        ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" resource $uri --output table
    fi
    
    echo "Testing Resource: $uri (JSON Output)"
    if [ "$uri" = "devices://dmi" ]; then
        ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" resource $uri --output json || echo "(Expected failure on some VMs)"
    else
        ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" resource $uri --output json
    fi
}

run_resource "os://hostname"
run_resource "os://release"
run_resource "os://uname"
run_resource "network://interfaces"
run_resource "network://routes"
run_resource "devices://usb"
run_resource "devices://pci"
run_resource "devices://dmi"
run_resource "kernel://modules"

UNPRIV_TOKEN="my-unprivileged-token-123"
echo "Testing unpriviliged user resource access (should fail)..."
if ./linuxctl -token "$UNPRIV_TOKEN" -server "$DAEMON_URL" resource "devices://usb" --output json; then
    echo "❌ FAILED: unpriviliged user was able to access devices://usb"
    exit 1
else
    echo "✅ SUCCESS: unpriviliged user was correctly denied access to devices://usb"
fi

echo "✅ SUCCESS: linuxctl successfully dynamically executed all tools and resources over SSE with format parsers!"
# Clean up the binary
rm -f linuxctl
exit 0
