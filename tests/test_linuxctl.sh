#!/bin/bash
set -e
# noglob: run_tool holds a whole command in one unquoted string, so a pattern
# like "kube*" must stay literal instead of expanding against the current dir.
set -f

# Everything target-specific is an env var; the defaults are the local
# k8s stand (configs/daemon.yaml). Point it at a VPS stand, e.g. on the host:
#   DAEMON_URL=https://localhost:9092 MCP_CA_CERT=/etc/mcpd-docker/configs/tls/mcpd.crt \
#   TOKEN=... PRIV_TOKEN=... UNPRIV_TOKEN=... LINUXCTL=$(which linuxctl) \
#   SERVICE=ssh.service SVC_PATTERN='ssh*' CURL_URL=https://example.com \
#   LOGINS_MAY_FAIL=no bash test_linuxctl.sh
DAEMON_URL=${DAEMON_URL:-"http://localhost:9091"}
TOKEN=${TOKEN:-"my-test-token-123"}
PRIV_TOKEN=${PRIV_TOKEN:-"my-privileged-token-123"}
UNPRIV_TOKEN=${UNPRIV_TOKEN:-"my-unprivileged-token-123"}
SERVICE=${SERVICE:-"kubelet.service"}            # a unit that exists on the target
SVC_PATTERN=${SVC_PATTERN:-"kube*"}              # matches at least that unit's family
CURL_URL=${CURL_URL:-"http://127.0.0.1:9091/ping"}
# logs/logins needs last/lastb: the minimal local kind node has neither, a normal host does.
LOGINS_MAY_FAIL=${LOGINS_MAY_FAIL:-"yes"}
# Optional: a container mcpd may exec into (docker/exec grant) to check `exec docker <c> -- ...`.
DOCKER_EXEC_CONTAINER=${DOCKER_EXEC_CONTAINER:-""}

echo "Testing linuxctl CLI (verb/group grammar - see plan/linuxctl-redesign.md)..."

if [ -z "$LINUXCTL" ]; then
    # Ensure we are in the project root
    cd "$(dirname "$0")/.."
    echo "Building linuxctl..."
    go build -o linuxctl ./cmd/linuxctl
    LINUXCTL=./linuxctl
    trap 'rm -f linuxctl' EXIT
fi

# lc <token> <linuxctl args...>
lc() { "$LINUXCTL" -token "$1" -server "$DAEMON_URL" "${@:2}"; }

# Test ping command
echo "Running 'linuxctl ping'..."
OUTPUT=$(lc "$TOKEN" ping) || true
if ! echo "$OUTPUT" | grep -q "Successfully connected to mcpd daemon!"; then
    echo "❌ FAILED: linuxctl failed to ping the daemon at $DAEMON_URL."
    echo "Output: $OUTPUT"
    exit 1
fi

# Discover what this target actually has, instead of assuming $DISK / eth0.
DISK=$(lc "$PRIV_TOKEN" get disks --output json | python3 -c "
import json,sys
d=[b['name'] for b in json.load(sys.stdin)['blockdevices'] if b.get('type')=='disk']
print(d[0] if d else 'vda')")
IFACE=$(lc "$PRIV_TOKEN" get network interfaces --output json | python3 -c "
import json,sys
n=[i['name'] for i in json.load(sys.stdin) if i['name']!='lo']
print(n[0] if n else 'eth0')")
echo "Target: disk=$DISK interface=$IFACE service=$SERVICE"

echo "==========================================="
echo "   Testing Tools (JSON & Table outputs)    "
echo "==========================================="

run_tool() {
    local cmd="$1"
    local name="$2"
    local allow_fail="${3:-no}"
    local tok="${TOK:-$TOKEN}"

    for fmt in table wide yaml json; do
        echo "Testing Tool: $name ($fmt Output)"
        if [ "$allow_fail" = "yes" ]; then
            lc "$tok" $cmd --output "$fmt" || echo "(Expected failure - see call site comment)"
        else
            lc "$tok" $cmd --output "$fmt"
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
run_tool "get disks partitions $DISK" "disks/partitions (device filter)"
run_tool "get disks performance" "disks/performance (all devices)"
run_tool "get disks performance $DISK" "disks/performance (one device)"
run_tool "get disks health $DISK" "disks/health"
run_tool "get processes --limit 5" "processes/list (bare)"
run_tool "get network connections" "network/connections"
run_tool "get network nslookup google.com" "network/nslookup"
run_tool "get network curl $CURL_URL" "network/curl"
run_tool "get network arp" "network/arp"
run_tool "get network ping 127.0.0.1" "network/ping"
run_tool "get memory usage" "memory/usage"
run_tool "get cpu" "cpu/list (bare)"
run_tool "get cpu load-average" "cpu/load-average"
run_tool "get kernel sysctl net.ipv4.ip_forward" "kernel/system-control (read)"
run_tool "get logs dmesg --privileged true" "logs/dmesg"
run_tool "get logs journal --lines 3 --privileged true" "logs/journal-control"
run_tool "get logs journal --lines 3 --boot true --privileged true" "logs/journal-control (boot)"
# logs/logins wraps last/lastb: the minimal local kind node has neither, so
# there it is an expected failure (LOGINS_MAY_FAIL=yes, the default); on a
# normal host set LOGINS_MAY_FAIL=no and it must pass.
if [ "$LOGINS_MAY_FAIL" = "yes" ]; then
    run_tool "get logs logins --privileged true" "logs/logins" "yes"
else
    run_tool "get logs logins --privileged true" "logs/logins"
fi
run_tool "get system os-release" "system/os-release"
run_tool "get system packages" "system/packages"
run_tool "get users --min_uid 1000" "users/list"
# The pattern stays literal (set -f above): a bare * would expand against the cwd.
run_tool "get system services --pattern $SVC_PATTERN --privileged true" "services/list (via system group)"
# timers/list (FR-025): privileged so it also reaches the host bus from inside a container
TOK=$PRIV_TOKEN run_tool "get timers --privileged true" "timers/list"
TOK=$PRIV_TOKEN run_tool "get timers --pattern *.timer --active_state active --privileged true" "timers/list (filtered)"

echo "==========================================="
echo "   Testing get-one vs get-many, describe,  "
echo "   top, and mutations (not run via loop -  "
echo "   these have side effects or need special "
echo "   handling, not 4x-format repetition)     "
echo "==========================================="

echo "Testing: get processes <pid> (one result, not the bare many-result form)"
FIRST_PID=$(lc "$PRIV_TOKEN" get processes --limit 1 --output json | python3 -c "import json,sys; print(json.load(sys.stdin)[0]['pid'])")
lc "$PRIV_TOKEN" get processes "$FIRST_PID" --output json

echo "Testing: get processes top (fixed cpu+memory+processes recipe)"
lc "$PRIV_TOKEN" get processes top

echo "Testing: describe files /etc/hosts (aggregates stat+type)"
lc "$TOKEN" describe files /etc/hosts

echo "Testing: describe disks $DISK"
lc "$PRIV_TOKEN" describe disks $DISK

echo "Testing: describe processes <pid> (aggregates status+cmdline+limits, excludes environ)"
lc "$PRIV_TOKEN" describe processes "$FIRST_PID"

echo "Testing: describe network interfaces $IFACE"
lc "$PRIV_TOKEN" describe network interfaces $IFACE

echo "Testing: create/update files (mutations, scratch path only)"
lc "$TOKEN" create files /tmp/linuxctl-grammar-test.txt --content "hello"
lc "$TOKEN" update files /tmp/linuxctl-grammar-test.txt --content " world" --append true

echo "Testing: update kernel sysctl (no-op - sets to its own current value)"
CURRENT_IP_FORWARD=$(lc "$PRIV_TOKEN" get kernel sysctl net.ipv4.ip_forward | grep -o '[01]$')
lc "$PRIV_TOKEN" update kernel sysctl net.ipv4.ip_forward "$CURRENT_IP_FORWARD" --privileged true

echo "Testing: explain <group> meta-verb"
lc "$TOKEN" explain files
lc "$TOKEN" explain system

echo "Testing: tool <name> direct escape hatch (symmetric with resource <uri>)"
lc "$TOKEN" tool files/list --path /tmp

echo "Testing: get mcp-api <tools|resources|prompts|info> meta-group"
lc "$TOKEN" get mcp-api tools > /tmp/mcp_tools_out.txt
grep -q "files/list" /tmp/mcp_tools_out.txt || { echo "❌ FAILED: get mcp-api tools missing files/list"; exit 1; }
lc "$TOKEN" get mcp-api resources > /tmp/mcp_resources_out.txt
grep -q "os://uname" /tmp/mcp_resources_out.txt || { echo "❌ FAILED: get mcp-api resources missing os://uname"; exit 1; }
# prompts is a real MCP capability mcpd doesn't implement - this must report
# that plainly, not silently succeed with an empty list or crash.
lc "$TOKEN" get mcp-api prompts > /tmp/mcp_prompts_out.txt
grep -q "does not implement" /tmp/mcp_prompts_out.txt || { echo "❌ FAILED: get mcp-api prompts did not report the missing capability"; exit 1; }
lc "$TOKEN" get mcp-api info > /tmp/mcp_info_out.txt
grep -q "protocolVersion" /tmp/mcp_info_out.txt || { echo "❌ FAILED: get mcp-api info missing protocolVersion"; exit 1; }
echo "✅ get mcp-api tools/resources/prompts/info all correct"

echo "==========================================="
echo "  Testing Resources (JSON & Table outputs) "
echo "==========================================="

run_resource() {
    local uri="$1"

    echo "Testing Resource: $uri (Table Output)"
    if [ "$uri" = "devices://dmi" ]; then
        lc "$TOKEN" resource $uri --output table || echo "(Expected failure on some VMs)"
    else
        lc "$TOKEN" resource $uri --output table
    fi

    echo "Testing Resource: $uri (JSON Output)"
    if [ "$uri" = "devices://dmi" ]; then
        lc "$TOKEN" resource $uri --output json || echo "(Expected failure on some VMs)"
    else
        lc "$TOKEN" resource $uri --output json
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
run_resource "network://interfaces/$IFACE"
run_resource "file:///etc/hosts"
run_resource "file:///etc/hosts/stat"
run_resource "file:///etc/hosts/type"
run_resource "devices://usb"
run_resource "devices://pci"
run_resource "devices://dmi"
run_resource "kernel://modules"

echo "Testing unpriviliged user resource access (should fail)..."
if lc "$UNPRIV_TOKEN" resource "devices://usb" --output json; then
    echo "❌ FAILED: unpriviliged user was able to access devices://usb"
    exit 1
else
    echo "✅ SUCCESS: unpriviliged user was correctly denied access to devices://usb"
fi

# service://{name}/status needs the host systemd dbus socket, only reachable
# privileged - testuser has no service:// grant in mcp-sudo.yaml (only the
# "privileged" reference account does), so this checks it with that token
# rather than expanding testuser's grants just for test coverage.
echo "Testing Resource: service://$SERVICE/status (privileged token, JSON Output)"
lc "$PRIV_TOKEN" resource "service://$SERVICE/status" --output json

# Docker group (FR-011..FR-020): every tool is listed to every user, so this
# runs whenever the privileged user is granted docker/containers; otherwise it
# reports that it was skipped.
echo "==========================================="
echo "  Testing docker group (read + FR-015/16/18)"
echo "==========================================="
if lc "$PRIV_TOKEN" get docker containers >/dev/null 2>&1; then
    for t in "get docker containers" "get docker networks" "get docker volumes" "get docker images"; do
        TOK=$PRIV_TOKEN run_tool "$t" "docker: $t"
    done
    echo "Testing: get docker network bridge (FR-015: reads the template, not docker/containers)"
    lc "$PRIV_TOKEN" get docker network bridge | grep -q '"Name": "bridge"' || { echo "❌ FAILED: get docker network bridge did not return the bridge network"; exit 1; }
    echo "Testing: get docker networkz (FR-015: an unknown keyword is an error)"
    if lc "$PRIV_TOKEN" get docker networkz >/dev/null 2>&1; then
        echo "❌ FAILED: get docker networkz exited 0"; exit 1
    fi
    echo "Testing: get docker network bridge extra-word (FR-016: extra word is reported)"
    lc "$PRIV_TOKEN" get docker network bridge extra-word 2>&1 >/dev/null | grep -q "extra argument" || { echo "❌ FAILED: no extra-argument warning"; exit 1; }
    if [ -n "$DOCKER_EXEC_CONTAINER" ]; then
        echo "Testing: exec docker $DOCKER_EXEC_CONTAINER -- echo hello (FR-018)"
        lc "$PRIV_TOKEN" exec docker "$DOCKER_EXEC_CONTAINER" -- echo hello | grep -q "^hello$" || { echo "❌ FAILED: exec after -- did not print hello"; exit 1; }
    else
        echo "(skipped: exec check - set DOCKER_EXEC_CONTAINER to a container the privileged user may exec into)"
    fi
    echo "Testing: ungranted user gets a clear refusal (FR-020)"
    if OUT=$(lc "$UNPRIV_TOKEN" get docker containers 2>&1); then
        echo "(unprivileged user holds a docker grant on this target - refusal not checked)"
    else
        echo "$OUT" | grep -q "not authorized" || { echo "❌ FAILED: refusal does not say not authorized: $OUT"; exit 1; }
        echo "✅ refused: $(echo "$OUT" | head -c 100)"
    fi
else
    echo "(skipped: the privileged user has no docker grant on this target, or docker is not there)"
fi

echo "✅ SUCCESS: linuxctl successfully dynamically executed all tools and resources over SSE with format parsers!"
exit 0
