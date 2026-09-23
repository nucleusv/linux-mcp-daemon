#!/usr/bin/env bash
# Checks that every MCP tool registered in internal/rpc/tools.go is listed
# everywhere the docs list tools by hand - and that nothing listed there is
# stale. tools/list itself is generated from the schema, so it can't drift;
# these hand-written lists can.
#
#   1. a page per tool:       docs/website/docs/mcp-api/tools/<group>/<command>.md
#   2. the MCP API overview:  docs/website/docs/mcp-api/overview.md
#   3. the reference grants:  the `privileged` user in configs/mcp-sudo.yaml
#                             (CLAUDE.md: it lists every tool the daemon exposes)
set -euo pipefail
cd "$(dirname "$0")/.."

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

grep -o '"name": *"[a-z-]*/[a-z-]*"' internal/rpc/tools.go \
    | grep -o '[a-z-]*/[a-z-]*' | sort -u > "$tmp/code"

find docs/website/docs/mcp-api/tools -name '*.md' \
    | sed -E 's#.*/tools/([^/]+)/([^/]+)\.md#\1/\2#' | sort -u > "$tmp/pages"

grep -o '\[`[a-z-]*/[a-z-]*`\]' docs/website/docs/mcp-api/overview.md \
    | tr -d '[]`' | sort -u > "$tmp/overview"

# Tool keys under the `privileged:` user's tools block in mcp-sudo.yaml.
awk '
    /^  privileged:/ { inuser=1; next }
    /^  [a-z]/       { inuser=0 }
    inuser && /^      tools:/ { intools=1; next }
    inuser && /^      resources:/ { intools=0 }
    inuser && intools && /^        [a-z-]+\/[a-z-]+:/ { gsub(/[ :]/, ""); print }
' configs/mcp-sudo.yaml | sort -u > "$tmp/sudo"

fail=0
report() { # report <description> <file with names>
    if [ -s "$2" ]; then
        echo "❌ $1:"
        sed 's/^/     /' "$2"
        fail=1
    fi
}

comm -23 "$tmp/code" "$tmp/pages"    > "$tmp/x"; report "tools without a docs page (docs/website/docs/mcp-api/tools/<group>/<command>.md)" "$tmp/x"
comm -23 "$tmp/code" "$tmp/overview" > "$tmp/x"; report "tools missing from docs/website/docs/mcp-api/overview.md" "$tmp/x"
comm -23 "$tmp/code" "$tmp/sudo"     > "$tmp/x"; report "tools missing from the privileged block of configs/mcp-sudo.yaml" "$tmp/x"
comm -13 "$tmp/code" "$tmp/pages"    > "$tmp/x"; report "docs pages for tools that no longer exist" "$tmp/x"
comm -13 "$tmp/code" "$tmp/overview" > "$tmp/x"; report "overview entries for tools that no longer exist" "$tmp/x"

if [ "$fail" -eq 0 ]; then
    echo "✅ All $(wc -l < "$tmp/code" | tr -d ' ') tools have a docs page, an overview entry and a privileged grant."
fi
exit "$fail"
