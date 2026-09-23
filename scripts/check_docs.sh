#!/usr/bin/env bash
# Checks that every MCP object the daemon registers - tools, static
# resources and resource templates - is listed everywhere the docs and the
# reference config list them by hand, and that nothing listed there is stale.
# tools/list, resources/list and resources/templates/list are generated from
# code and can't drift; these hand-written lists can.
#
# Sources of truth (code):
#   tools            "name": "<group>/<command>"   in internal/rpc/tools.go
#   resources        "uri": "scheme://..."          in internal/rpc/resources.go
#   templates        "uriTemplate": "scheme://..."  in internal/rpc/resources.go
#   template grants  what each template handler in internal/resources/templates/
#                    checks (CanReadResourceAsRoot "<scheme>") and which worker
#                    tools it spawns - both need a grant for privileged reads
#                    (ARCHITECTURE.md: "privileged resource reads need TWO grants")
#
# Checked against:
#   pages     docs/website/docs/mcp-api/{tools/<group>/<command>.md,
#             resources/*.md (**URI**: `...`), resource-templates/*.md (**URI Template**: `...`)}
#   overview  docs/website/docs/mcp-api/overview.md, including that each link
#             points at an existing page
#   grants    the reference `privileged` user in configs/mcp-sudo.yaml, which
#             CLAUDE.md says lists every tool and resource the daemon exposes
set -euo pipefail
cd "$(dirname "$0")/.."

DOCS=docs/website/docs/mcp-api
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

fail=0
report() { # report <description> <file with offending names>
    if [ -s "$2" ]; then
        echo "❌ $1:"
        sed 's/^/     /' "$2"
        fail=1
    fi
}
missing() { comm -23 "$1" "$2" > "$tmp/x"; report "$3" "$tmp/x"; }  # in $1, not in $2
stale()   { comm -13 "$1" "$2" > "$tmp/x"; report "$3" "$tmp/x"; }  # in $2, not in $1

# ------------------------------------------------------------------ code
grep -o '"name": *"[a-z-]*/[a-z-]*"' internal/rpc/tools.go \
    | grep -o '[a-z-]*/[a-z-]*' | sort -u > "$tmp/code_tools"
grep -o '"uri": *"[^"]*"' internal/rpc/resources.go \
    | sed -E 's/.*"uri": *"([^"]*)"/\1/' | sort -u > "$tmp/code_res"
grep -o '"uriTemplate": *"[^"]*"' internal/rpc/resources.go \
    | sed -E 's/.*"uriTemplate": *"([^"]*)"/\1/' | sort -u > "$tmp/code_tpl"

# Grants the template handlers rely on for privileged reads: the resource
# scheme they check, and every internal worker tool they may spawn.
: > "$tmp/need_res_grants"; : > "$tmp/need_tool_grants"
for f in internal/resources/templates/*/*.go; do
    case "$f" in *_test.go) continue ;; esac
    scheme="$(grep -o 'CanReadResourceAsRoot([^,]*, *"[a-z]*://"' "$f" | grep -o '"[a-z]*://"' | tr -d '"' || true)"
    [ -n "$scheme" ] || continue   # never privileged: needs no grant
    echo "$scheme" >> "$tmp/need_res_grants"
    # Worker names look like tools: "<group>/<command>" with a real tool
    # group (not imports such as "encoding/json" or MIME types).
    grep -o '"[a-z-]*/[a-z-]*"' "$f" | tr -d '"' \
        | awk -F/ 'NR==FNR { g[$1]=1; next } ($1 in g) && $2 != ""' "$tmp/code_tools" - >> "$tmp/need_tool_grants"
done
sort -u -o "$tmp/need_res_grants" "$tmp/need_res_grants"
sort -u -o "$tmp/need_tool_grants" "$tmp/need_tool_grants"

# ------------------------------------------------------------------ docs
find "$DOCS/tools" -name '*.md' \
    | sed -E 's#.*/tools/([^/]+)/([^/]+)\.md#\1/\2#' | sort -u > "$tmp/page_tools"
for f in "$DOCS"/resources/*.md; do
    grep -m1 -o '\*\*URI\*\*: `[^`]*`' "$f" | sed -E 's/.*`([^`]*)`/\1/'
done | sort -u > "$tmp/page_res"
for f in "$DOCS"/resource-templates/*.md; do
    grep -m1 -o '\*\*URI Template\*\*: `[^`]*`' "$f" | sed -E 's/.*`([^`]*)`/\1/'
done | sort -u > "$tmp/page_tpl"

OV="$DOCS/overview.md"
grep -o '\[`[a-z-]*/[a-z-]*`\](\./tools/' "$OV" | grep -o '`[^`]*`' | tr -d '`' | sort -u > "$tmp/ov_tools"
grep -o '\[`[^`]*`\](\./resources/' "$OV" | grep -o '`[^`]*`' | tr -d '`' | sort -u > "$tmp/ov_res"
grep -o '\[`[^`]*`\](\./resource-templates/' "$OV" | grep -o '`[^`]*`' | tr -d '`' | sort -u > "$tmp/ov_tpl"

# Every overview link must point at a page that exists.
grep -o '](\./\(tools\|resources\|resource-templates\)/[^)]*)' "$OV" | sed -E 's/^\]\(\.\/(.*)\)$/\1/' | sort -u \
    | while IFS= read -r link; do [ -f "$DOCS/$link.md" ] || echo "$link"; done > "$tmp/x"
report "overview links to pages that don't exist" "$tmp/x"

# ---------------------------------------------------------- mcp-sudo.yaml
# Keys of the `privileged` user's tools: and resources: blocks.
sudo_keys() { # sudo_keys tools|resources
    awk -v want="$1" '
        /^  privileged:/ { inuser=1; next }
        /^  [a-z]/       { inuser=0 }
        inuser && /^      tools:/     { block="tools"; next }
        inuser && /^      resources:/ { block="resources"; next }
        inuser && block==want && /^        [^ #]/ {
            key=$0; sub(/^ +/, "", key); sub(/:[ ]*$/, "", key); gsub(/"/, "", key); print key
        }
    ' configs/mcp-sudo.yaml | sort -u
}
sudo_keys tools > "$tmp/sudo_tools"
sudo_keys resources > "$tmp/sudo_res"

# --------------------------------------------------------------- checks
echo "== tools ($(wc -l < "$tmp/code_tools" | tr -d ' '))"
missing "$tmp/code_tools" "$tmp/page_tools" "tools without a docs page ($DOCS/tools/<group>/<command>.md)"
missing "$tmp/code_tools" "$tmp/ov_tools"   "tools missing from $OV"
missing "$tmp/code_tools" "$tmp/sudo_tools" "tools missing from the privileged block of configs/mcp-sudo.yaml"
stale   "$tmp/code_tools" "$tmp/page_tools" "docs pages for tools that no longer exist"
stale   "$tmp/code_tools" "$tmp/ov_tools"   "overview entries for tools that no longer exist"

echo "== resources ($(wc -l < "$tmp/code_res" | tr -d ' '))"
missing "$tmp/code_res" "$tmp/page_res" "resources without a docs page ($DOCS/resources/*.md with **URI**: \`...\`)"
missing "$tmp/code_res" "$tmp/ov_res"   "resources missing from $OV"
missing "$tmp/code_res" "$tmp/sudo_res" "resources missing from the privileged block of configs/mcp-sudo.yaml"
stale   "$tmp/code_res" "$tmp/page_res" "docs pages for resources that no longer exist"
stale   "$tmp/code_res" "$tmp/ov_res"   "overview entries for resources that no longer exist"

echo "== resource templates ($(wc -l < "$tmp/code_tpl" | tr -d ' '))"
missing "$tmp/code_tpl" "$tmp/page_tpl" "templates without a docs page ($DOCS/resource-templates/*.md with **URI Template**: \`...\`)"
missing "$tmp/code_tpl" "$tmp/ov_tpl"   "templates missing from $OV"
stale   "$tmp/code_tpl" "$tmp/page_tpl" "docs pages for templates that no longer exist"
stale   "$tmp/code_tpl" "$tmp/ov_tpl"   "overview entries for templates that no longer exist"
missing "$tmp/need_res_grants"  "$tmp/sudo_res"   "template schemes the privileged block doesn't grant (resources:)"
missing "$tmp/need_tool_grants" "$tmp/sudo_tools" "worker tools behind privileged templates the privileged block doesn't grant (tools:) - the second of the two grants a privileged resource read needs"

echo "== stale grants"
# Resource grants: a static URI, or a template scheme ("file://").
cat "$tmp/code_res" "$tmp/need_res_grants" | sort -u > "$tmp/valid_res"
stale "$tmp/valid_res" "$tmp/sudo_res" "privileged resource grants for resources that no longer exist"
# Tool grants: a registered tool, or an internal worker behind a template.
cat "$tmp/code_tools" "$tmp/need_tool_grants" | sort -u > "$tmp/valid_tools"
stale "$tmp/valid_tools" "$tmp/sudo_tools" "privileged tool grants for tools that no longer exist"

if grep -q '"prompts/list"' internal/rpc/*.go 2>/dev/null; then
    echo "note: prompts/list is implemented - extend this script to check prompts too"
fi

if [ "$fail" -eq 0 ]; then
    echo "✅ Every tool, resource and resource template has a docs page and an overview entry, every grant a privileged read needs is in the reference config, and nothing listed is stale."
fi
exit "$fail"
