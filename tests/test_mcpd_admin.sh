#!/bin/bash
set -e

# linuxctl's "mcpd" group is local-only (no daemon involved), so this test
# never touches DAEMON_URL/TOKEN - it runs entirely against a scratch copy
# of configs/, matching the pattern used to verify this feature live during
# development (see docs/website/docs/configuration/daemon.md).

echo "Testing linuxctl mcpd (local-only user/token admin)..."

cd "$(dirname "$0")/.."

echo "Building linuxctl..."
go build -o linuxctl ./cmd/linuxctl

SCRATCH=$(mktemp -d)
cleanup() {
    rm -rf "$SCRATCH"
    rm -f linuxctl
}
trap cleanup EXIT

cp -r configs "$SCRATCH/configs"

fail() {
    echo "❌ FAILED: $1"
    exit 1
}

echo "--- create ---"
OUT=$(./linuxctl create mcpd user testadmin --config-path "$SCRATCH/configs")
echo "$OUT" | grep -q 'Created mcpd user "testadmin"' || fail "create did not report success"
TOKEN=$(echo "$OUT" | sed -n '/shown once/{n;p;}' | tr -d ' ')
[ -n "$TOKEN" ] || fail "create did not print a token"
grep -q "token_hash" "$SCRATCH/configs/daemon.yaml" || fail "daemon.yaml missing token_hash after create"
grep -q "testadmin:" "$SCRATCH/configs/mcp-sudo.yaml" || fail "mcp-sudo.yaml missing fresh block after create"

echo "--- duplicate create is rejected ---"
if ./linuxctl create mcpd user testadmin --config-path "$SCRATCH/configs" 2>/dev/null; then
    fail "create allowed a duplicate username"
fi
echo "✅ duplicate create correctly rejected"

echo "--- list shows it ---"
./linuxctl list mcpd users --config-path "$SCRATCH/configs" | grep -q "testadmin" || fail "list did not show the new user"

echo "--- describe shows fresh empty grants ---"
DESC=$(./linuxctl describe mcpd user testadmin --config-path "$SCRATCH/configs")
echo "$DESC" | grep -q "tools: {}" || fail "describe did not show empty tools grant"

echo "--- update rotates the token ---"
OUT2=$(./linuxctl update mcpd user testadmin --config-path "$SCRATCH/configs")
NEWTOKEN=$(echo "$OUT2" | sed -n '/shown once/{n;p;}' | tr -d ' ')
[ -n "$NEWTOKEN" ] || fail "update did not print a new token"
[ "$NEWTOKEN" != "$TOKEN" ] || fail "update produced the same token as create"

echo "--- the core bug this feature fixes: a stale mcp-sudo.yaml block must never survive delete+recreate ---"
# Simulate a manual edit that grants something dangerous, matching the real
# incident this feature was built to prevent.
python3 -c "
p = '$SCRATCH/configs/mcp-sudo.yaml'
s = open(p).read()
s = s.replace('testadmin:\n    privileged:\n      tools: {}', 'testadmin:\n    privileged:\n      tools:\n        kernel/system-control:\n          allowed: true')
open(p, 'w').write(s)
"
grep -q "kernel/system-control" "$SCRATCH/configs/mcp-sudo.yaml" || fail "test setup: stale grant injection failed"
# Only remove the daemon.yaml entry (the actual mistake in the real incident:
# a partial manual edit touching one file but not the other).
python3 -c "
import re
p = '$SCRATCH/configs/daemon.yaml'
s = open(p).read()
s = re.sub(r'\n  - username: testadmin\n.*?\n.*?\n.*?\n', '\n', s, flags=re.DOTALL)
open(p, 'w').write(s)
"
grep -q "testadmin" "$SCRATCH/configs/daemon.yaml" && fail "test setup: daemon.yaml entry removal failed"
./linuxctl create mcpd user testadmin --config-path "$SCRATCH/configs" > /tmp/recreate_out.txt
grep -q "stale entry" /tmp/recreate_out.txt || fail "create did not warn about the stale mcp-sudo.yaml entry"
RECREATE_DESC=$(./linuxctl describe mcpd user testadmin --config-path "$SCRATCH/configs")
if echo "$RECREATE_DESC" | grep -q "kernel/system-control"; then
    fail "recreated user inherited the stale privileged grant - THE CORE BUG IS BACK"
fi
echo "✅ recreated user correctly got a fresh, empty grant block (no inherited privileges)"

echo "--- delete removes from both files ---"
./linuxctl delete mcpd user testadmin --config-path "$SCRATCH/configs" || fail "delete failed"
grep -q "testadmin" "$SCRATCH/configs/daemon.yaml" && fail "daemon.yaml still has testadmin after delete"
grep -q "testadmin:" "$SCRATCH/configs/mcp-sudo.yaml" && fail "mcp-sudo.yaml still has testadmin after delete"

echo "--- comments in the original files are preserved (yaml.Node surgical edit, not a lossy struct re-marshal) ---"
grep -q "This daemon runs inside a Kubernetes pod" "$SCRATCH/configs/daemon.yaml" || fail "daemon.yaml lost its worker: comment"
grep -q "grant everything.*means an empty-string prefix" "$SCRATCH/configs/mcp-sudo.yaml" || fail "mcp-sudo.yaml lost its resources comment"

echo "✅ SUCCESS: linuxctl mcpd admin commands work correctly end-to-end, including the delete+recreate safety guarantee!"
exit 0
