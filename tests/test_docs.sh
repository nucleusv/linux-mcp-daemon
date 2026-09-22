#!/bin/bash

DAEMON_URL=${DAEMON_URL:-"http://localhost:9091"}

echo "Waiting for Documentation server to come online..."

# Dynamically define all expected pages by scanning the source directory
EXPECTED_PAGES=("/docs/")
# Ensure we are in the project root
cd "$(dirname "$0")/.."
while IFS= read -r file; do
    # Remove docs/website/docs/ prefix
    rel_path="${file#docs/website/docs/}"
    # Remove .md suffix
    route_path="${rel_path%.md}"
    
    
    # Docusaurus maps index.md to the root of its folder
    if [[ "$route_path" == *"/index" ]]; then
        route_path="${route_path%/index}"
    fi
    
    # Handle root index.md which becomes empty string
    if [ "$route_path" == "index" ] || [ "$route_path" == "" ]; then
        continue
    fi
    
    EXPECTED_PAGES+=("/docs/$route_path/")
done < <(find docs/website/docs -type f -name "*.md")

MAX_RETRIES=10
RETRY_COUNT=0
SERVER_ONLINE=false

# First, wait for the root server to come online
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    HTTP_STATUS=$(curl -s -L -o /dev/null -w "%{http_code}" "$DAEMON_URL/docs/" || echo "000")
    if [ "$HTTP_STATUS" -eq 200 ]; then
        SERVER_ONLINE=true
        break
    fi
    
    echo "Server returned HTTP $HTTP_STATUS. Retrying in 2 seconds..."
    sleep 2
    RETRY_COUNT=$((RETRY_COUNT + 1))
done

if [ "$SERVER_ONLINE" = false ]; then
    echo "❌ FAILED: Documentation server never came online after $MAX_RETRIES attempts."
    exit 1
fi

echo "✅ Server is online! Validating individual documentation pages..."

FAILED=0

for PAGE in "${EXPECTED_PAGES[@]}"; do
    # URL encode spaces
    ENCODED_PAGE="${PAGE// /%20}"
    STATUS=$(curl -s -L -o /dev/null -w "%{http_code}" "$DAEMON_URL$ENCODED_PAGE" || echo "000")
    if [ "$STATUS" -eq 200 ]; then
        echo "  [OK] $PAGE"
    else
        echo "  [ERROR] $PAGE returned HTTP $STATUS"
        FAILED=1
    fi
done

if [ "$FAILED" -eq 1 ]; then
    echo "❌ FAILED: One or more documentation pages did not return HTTP 200 OK."
    exit 1
else
    echo "✅ SUCCESS: All expected documentation pages are rendering properly!"
    exit 0
fi
