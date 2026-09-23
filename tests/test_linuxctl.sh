#!/bin/bash
set -e

DAEMON_URL=${DAEMON_URL:-"http://localhost:9091"}
TOKEN=${TOKEN:-"my-test-token-123"}
PRIV_TOKEN="my-privileged-token-123"

echo "Testing linuxctl CLI (verb/group grammar - see plan/linuxctl-redesign.md)..."

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

run_tool "get auth sudo-rules" "auth/sudo-rules"
run_tool "get files list /var/log --privileged true" "files/list"
run_tool "get files find /etc --name hosts*" "files/find"
run_tool "get files filetype /etc/hosts" "files/filetype"
run_tool "get disks usage /var/log --privileged true" "disks/usage"
run_tool "get disks free / --privileged true" "disks/free"
run_tool "get disks" "disks/list (bare)"
run_tool "get disks mounts" "disks/mounts"
run_tool "get disks partitions" "disks/partitions"
run_tool "get disks partitions vda" "disks/partitions (device filter)"
run_tool "get disks performance" "disks/performance (all devices)"
run_tool "get disks performance vda" "disks/performance (one device)"
run_tool "get disks health vda" "disks/health"
run_tool "get processes --limit 5" "processes/list (bare)"
run_tool "get network connections" "network/connections"
run_tool "get network nslookup google.com" "network/nslookup"
run_tool "get network curl http://127.0.0.1:9091/ping" "network/curl"
run_tool "get network arp" "network/arp"
run_tool "get network ping 127.0.0.1" "network/ping"
run_tool "get memory usage" "memory/usage"
run_tool "get cpu" "cpu/list (bare)"
run_tool "get cpu load-average" "cpu/load-average"
run_tool "get kernel sysctl net.ipv4.ip_forward" "kernel/system-control (read)"
run_tool "get logs dmesg --privileged true" "logs/dmesg"
run_tool "get logs journal --lines 3 --privileged true" "logs/journal-control"
run_tool "get logs journal --lines 3 --boot true --privileged true" "logs/journal-control (boot)"
# allow_fail=yes: this environment's kind node has no last/lastb binaries at
# all (a minimal LinuxKit node with no login mechanism) - see
# investigations/README.md. Expected to fail here; would work on a normal host.
run_tool "get logs logins --privileged true" "logs/logins" "yes"
run_tool "get system os-release" "system/os-release"
run_tool "get system packages" "system/packages"
run_tool "get users --min_uid 1000" "users/list"
# --pattern kube* not --pattern * : run_tool uses $cmd unquoted (word
# splitting is required, since it's one string holding a whole multi-token
# command) - a bare "*" is a real shell glob against the CURRENT DIRECTORY
# at expansion time, not a literal token, and silently expands to every file
# in the repo root as separate ignored arguments. "kube*" matches nothing
# locally so bash leaves it literal, same trick the daemon's own live k8s
# testing already relied on elsewhere this session.
run_tool "get system services --pattern kube* --privileged true" "services/list (via system group)"

echo "==========================================="
echo "   Testing get-one vs get-many, describe,  "
echo "   top, and mutations (not run via loop -  "
echo "   these have side effects or need special "
echo "   handling, not 4x-format repetition)     "
echo "==========================================="

echo "Testing: get processes <pid> (one result, not the bare many-result form)"
FIRST_PID=$(./linuxctl -token "$PRIV_TOKEN" -server "$DAEMON_URL" get processes --limit 1 --output json | python3 -c "import json,sys; print(json.load(sys.stdin)[0]['pid'])")
./linuxctl -token "$PRIV_TOKEN" -server "$DAEMON_URL" get processes "$FIRST_PID" --output json

echo "Testing: get processes top (fixed cpu+memory+processes recipe)"
./linuxctl -token "$PRIV_TOKEN" -server "$DAEMON_URL" get processes top

echo "Testing: describe files /etc/hosts (aggregates stat+type)"
./linuxctl -token "$TOKEN" -server "$DAEMON_URL" describe files /etc/hosts

echo "Testing: describe disks vda"
./linuxctl -token "$PRIV_TOKEN" -server "$DAEMON_URL" describe disks vda

echo "Testing: describe processes <pid> (aggregates status+cmdline+limits, excludes environ)"
./linuxctl -token "$PRIV_TOKEN" -server "$DAEMON_URL" describe processes "$FIRST_PID"

echo "Testing: describe network interfaces eth0"
./linuxctl -token "$PRIV_TOKEN" -server "$DAEMON_URL" describe network interfaces eth0

echo "Testing: create/update files (mutations, scratch path only)"
./linuxctl -token "$TOKEN" -server "$DAEMON_URL" create files /tmp/linuxctl-grammar-test.txt --content "hello"
./linuxctl -token "$TOKEN" -server "$DAEMON_URL" update files /tmp/linuxctl-grammar-test.txt --content " world" --append true

echo "Testing: update kernel sysctl (no-op - sets to its own current value)"
CURRENT_IP_FORWARD=$(./linuxctl -token "$PRIV_TOKEN" -server "$DAEMON_URL" get kernel sysctl net.ipv4.ip_forward | grep -o '[01]$')
./linuxctl -token "$PRIV_TOKEN" -server "$DAEMON_URL" update kernel sysctl net.ipv4.ip_forward "$CURRENT_IP_FORWARD" --privileged true

echo "Testing: explain <group> meta-verb"
./linuxctl -token "$TOKEN" -server "$DAEMON_URL" explain files
./linuxctl -token "$TOKEN" -server "$DAEMON_URL" explain system

echo "Testing: tool <name> direct escape hatch (symmetric with resource <uri>)"
./linuxctl -token "$TOKEN" -server "$DAEMON_URL" tool files/list --path /tmp

echo "Testing: get mcp <tools|resources|prompts|info> meta-group"
./linuxctl -token "$TOKEN" -server "$DAEMON_URL" get mcp tools > /tmp/mcp_tools_out.txt
grep -q "files/list" /tmp/mcp_tools_out.txt || { echo "❌ FAILED: get mcp tools missing files/list"; exit 1; }
./linuxctl -token "$TOKEN" -server "$DAEMON_URL" get mcp resources > /tmp/mcp_resources_out.txt
grep -q "os://uname" /tmp/mcp_resources_out.txt || { echo "❌ FAILED: get mcp resources missing os://uname"; exit 1; }
# prompts is a real MCP capability mcpd doesn't implement - this must report
# that plainly, not silently succeed with an empty list or crash.
./linuxctl -token "$TOKEN" -server "$DAEMON_URL" get mcp prompts > /tmp/mcp_prompts_out.txt
grep -q "does not implement" /tmp/mcp_prompts_out.txt || { echo "❌ FAILED: get mcp prompts did not report the missing capability"; exit 1; }
./linuxctl -token "$TOKEN" -server "$DAEMON_URL" get mcp info > /tmp/mcp_info_out.txt
grep -q "protocolVersion" /tmp/mcp_info_out.txt || { echo "❌ FAILED: get mcp info missing protocolVersion"; exit 1; }
echo "✅ get mcp tools/resources/prompts/info all correct"

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
run_resource "network://interfaces/eth0"
run_resource "file:///etc/hosts"
run_resource "file:///etc/hosts/stat"
run_resource "file:///etc/hosts/type"
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

# service://{name}/status needs the host systemd dbus socket, only reachable
# privileged - testuser has no service:// grant in mcp-sudo.yaml (only the
# "privileged" reference account does), so this checks it with that token
# rather than expanding testuser's grants just for test coverage.
echo "Testing Resource: service://kubelet.service/status (privileged token, JSON Output)"
./linuxctl -token "$PRIV_TOKEN" -server "$DAEMON_URL" resource "service://kubelet.service/status" --output json

echo "✅ SUCCESS: linuxctl successfully dynamically executed all tools and resources over SSE with format parsers!"
# Clean up the binary
rm -f linuxctl
exit 0
