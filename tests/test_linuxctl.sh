#!/bin/bash
set -e

DAEMON_URL=${DAEMON_URL:-"http://localhost:9091"}
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
    local allow_fail="${3:-no}"

    for fmt in table wide yaml json; do
        echo "Testing Tool: $name ($fmt Output)"
        if [ "$allow_fail" = "yes" ]; then
            ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" $cmd --output "$fmt" || echo "(Expected failure - see call site comment)"
        else
            ./linuxctl -token "$TOKEN" -server "$DAEMON_URL" $cmd --output "$fmt"
        fi
    done

    # Each of the 4 calls above involves its own SSE handshake + tools/call
    # round trip; running through ~30 tools back to back easily exceeds the
    # daemon's per-user rate limiter burst (configs/daemon.yaml
    # rate_limits.default_burst) before its token bucket refills. This pause
    # keeps the suite under that limit rather than loosening it for test
    # convenience.
    sleep 0.3
}

run_tool "auth sudo-rules" "auth/sudo-rules"
run_tool "files list --path /var/log --privileged true" "files/list"
run_tool "files find --path /etc --name hosts*" "files/find"
run_tool "files filetype --path /etc/hosts" "files/filetype"
run_tool "disks usage --path /var/log --privileged true" "disks/usage"
run_tool "disks free --path / --privileged true" "disks/free"
run_tool "disks list" "disks/list"
run_tool "disks mounts" "disks/mounts"
run_tool "disks partitions" "disks/partitions"
run_tool "disks performance" "disks/performance"
run_tool "processes list --limit 5" "processes/list"
run_tool "network connections" "network/connections"
run_tool "network nslookup --host google.com" "network/nslookup"
run_tool "network curl --url http://127.0.0.1:9091/ping" "network/curl"
run_tool "network arp" "network/arp"
run_tool "network ping --host 127.0.0.1" "network/ping"
run_tool "memory usage" "memory/usage"
run_tool "cpu list" "cpu/list"
run_tool "cpu load-average" "cpu/load-average"
run_tool "kernel system-control --key net.ipv4.ip_forward" "kernel/system-control"
run_tool "logs dmesg --privileged true" "logs/dmesg"
run_tool "logs journal-control --lines 3 --privileged true" "logs/journal-control"
run_tool "logs journal-control --lines 3 --boot true --privileged true" "logs/journal-control (boot)"
# allow_fail=yes: this environment's kind node has no last/lastb binaries at
# all (a minimal LinuxKit node with no login mechanism) - see
# investigations/README.md. Expected to fail here; would work on a normal host.
run_tool "logs logins --privileged true" "logs/logins" "yes"
run_tool "system os-release" "system/os-release"
run_tool "system packages" "system/packages"
run_tool "users list --min-uid 1000" "users/list"
run_tool "services list --pattern *" "services/list"

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

    sleep 0.3
}

run_resource "system://hostname"
run_resource "system://timezone"
run_resource "system://locale"
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
