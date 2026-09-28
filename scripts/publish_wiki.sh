#!/bin/bash
# scripts/publish_wiki.sh
#
# Mirrors backlog/ into the GitHub wiki - one wiki page per ticket, plus a
# status-grouped board on Home.md. Every ticket in every status is public
# (backlog/README.md's statuses are all meant to be seen: new/in-progress/
# review/blocked are the live board, closed/rejected are its changelog).
#
# Ticket pages are the ticket's own markdown, byte for byte - no side
# rendering, matching backlog/README.md's "everything about the task lives
# in it" rule. Home.md keeps its existing docs-links section (hand-written,
# preserved below) and gets a generated "## Tasks" section appended after it.
#
# Usage: scripts/publish_wiki.sh [--dry-run]
#   --dry-run   write into ./wiki-preview/ instead of cloning/pushing the
#               real wiki repo - for checking output before it goes live.
set -euo pipefail
cd "$(dirname "$0")/.."

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

REPO="nucleusv/linux-mcp-daemon"

if $DRY_RUN; then
  WIKI_DIR="./wiki-preview"
  rm -rf "$WIKI_DIR"
  git clone --quiet --depth 1 "https://github.com/$REPO.wiki.git" "$WIKI_DIR"
else
  WIKI_DIR=$(mktemp -d)
  trap 'rm -rf "$WIKI_DIR"' EXIT
  git clone --quiet --depth 1 "https://x-access-token:${GITHUB_TOKEN}@github.com/$REPO.wiki.git" "$WIKI_DIR"
fi

# Every status is a board column, in this order - blocked/rejected carry a
# reason so they're worth surfacing even though they're not "in flight".
STATUSES=(new in-progress blocked review closed rejected)
declare -A STATUS_LABEL=(
  [new]="🆕 New" [in-progress]="🚧 In Progress" [blocked]="⛔ Blocked"
  [review]="👀 Review" [closed]="✅ Closed" [rejected]="🗑️ Rejected"
)

# Remove ticket pages the wiki has but backlog/ doesn't (closed-and-deleted,
# renamed) - never touch Home.md or anything git itself owns.
for f in "$WIKI_DIR"/FR-*.md; do
  [[ -e "$f" ]] || continue
  base=$(basename "$f")
  if ! find backlog -maxdepth 2 -name "$base" -print -quit | grep -q .; then
    rm "$f"
  fi
done

# Copy every ticket verbatim, and build the status-grouped board as we go.
BOARD=""
for status in "${STATUSES[@]}"; do
  BOARD+=$'\n'"### ${STATUS_LABEL[$status]}"$'\n\n'
  found=false
  for f in backlog/"$status"/FR-*.md; do
    [[ -e "$f" ]] || continue
    found=true
    base=$(basename "$f")
    cp "$f" "$WIKI_DIR/$base"
    title=$(grep -m1 '^# ' "$f" | sed 's/^# //')
    id=$(basename "$f" .md)
    BOARD+="- [$title](${id})"$'\n'
  done
  $found || BOARD+="_none_"$'\n'
done

# Home.md: keep the hand-written intro/quick-links section (everything
# above the "## Tasks" marker, or the whole file if the marker isn't there
# yet - first run) and replace everything from that marker down.
HOME="$WIKI_DIR/Home.md"
if [[ -f "$HOME" ]] && grep -q '^## Tasks' "$HOME"; then
  sed -n '1,/^## Tasks/{/^## Tasks/!p}' "$HOME" > "$HOME.new"
else
  cat > "$HOME.new" <<'INTRO'
# Linux MCPd documentation

📖 **The documentation lives at https://nucleusv.github.io/linux-mcp-daemon/**

The docs site is built from [`docs/website`](https://github.com/nucleusv/linux-mcp-daemon/tree/main/docs/website) in this repo, so it always matches the code.

## Quick links

- [Introduction](https://nucleusv.github.io/linux-mcp-daemon/intro/)
- [Architecture: ephemeral workers](https://nucleusv.github.io/linux-mcp-daemon/architecture/ephemeral-workers/)
- [AI agent configuration](https://nucleusv.github.io/linux-mcp-daemon/ai-agent-configuration/)
- Configuration: [daemon](https://nucleusv.github.io/linux-mcp-daemon/configuration/daemon/) · [mcp-sudo](https://nucleusv.github.io/linux-mcp-daemon/configuration/mcp-sudo/)
- [MCP API reference](https://nucleusv.github.io/linux-mcp-daemon/category/mcp-api/): every tool and resource, with curl examples
- [linuxctl CLI](https://nucleusv.github.io/linux-mcp-daemon/linuxctl/overview/) · [Command reference](https://nucleusv.github.io/linux-mcp-daemon/linuxctl/command-reference/) · [Shell autocompletion (bash/zsh)](https://nucleusv.github.io/linux-mcp-daemon/linuxctl/autocompletion/)

To fix or improve the docs, open a pull request against [`docs/website/docs/`](https://github.com/nucleusv/linux-mcp-daemon/tree/main/docs/website/docs).
INTRO
fi
{
  cat "$HOME.new"
  echo
  echo "## Tasks"
  echo
  echo "The task tracker - see [backlog/README.md](https://github.com/$REPO/blob/main/backlog/README.md) for what each status means. Generated from [\`backlog/\`](https://github.com/$REPO/tree/main/backlog) on every push that touches it - edit tickets there, not here."
  echo "$BOARD"
} > "$HOME"
rm -f "$HOME.new"

if $DRY_RUN; then
  echo "Dry run written to $WIKI_DIR - review it, then run without --dry-run."
  exit 0
fi

cd "$WIKI_DIR"
git add -A
if git diff --cached --quiet; then
  echo "No changes to publish."
  exit 0
fi
git -c user.name="github-actions[bot]" -c user.email="github-actions[bot]@users.noreply.github.com" \
  commit -m "Sync from backlog/ ($(cd - >/dev/null && git rev-parse --short HEAD))"
git push
