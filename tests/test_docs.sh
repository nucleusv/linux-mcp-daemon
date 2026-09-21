#!/bin/bash
set -e

DAEMON_URL=${DAEMON_URL:-"http://localhost:9090"}

echo "Waiting for Documentation server to come online..."

# Define all expected pages
EXPECTED_PAGES=(
    "/docs/"
    "/docs/intro/"
    "/docs/tools/list_directory/"
    "/docs/tools/get_disk_space/"
    "/docs/tools/get_disk_usage/"
    "/docs/architecture/ephemeral-workers/"
)

MAX_RETRIES=10
RETRY_COUNT=0
SERVER_ONLINE=false

# First, wait for the root server to come online
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    HTTP_STATUS=$(curl -s -L -o /dev/null -w "%{http_code}" "$DAEMON_URL/docs/")
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
    STATUS=$(curl -s -L -o /dev/null -w "%{http_code}" "$DAEMON_URL$PAGE")
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
