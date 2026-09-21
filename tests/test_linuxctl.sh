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
    
    echo "Testing Tool: $name (JSON Output)"
    ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" $cmd --output json
}

run_tool "auth/get-sudo-rules" "get_sudo_rules"
run_tool "files/list /var/log --privileged true" "list_files"
run_tool "disks/get-usage --path /var/log --privileged true" "get_usage"
run_tool "disks/get-free --path / --privileged true" "get_free"
run_tool "disks/get-blocks" "get_blocks"
run_tool "processes/list --limit 5" "list_processes"
run_tool "network/connections/list" "list_connections"
run_tool "network/nslookup --host 1.1.1.1" "nslookup"
run_tool "network/curl --url http://127.0.0.1:9090/ping" "curl"
run_tool "network/arp" "arp"
run_tool "network/ping --host 127.0.0.1" "ping"
run_tool "memory/usage" "memory_usage"
run_tool "cpu/get-info" "get_info"
run_tool "cpu/get-load-average" "get_load_average"

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

echo "✅ SUCCESS: linuxctl successfully dynamically executed all tools and resources over SSE with format parsers!"
# Clean up the binary
rm -f linuxctl
exit 0
