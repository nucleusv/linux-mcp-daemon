#!/bin/bash
set -e

DAEMON_URL=${DAEMON_URL:-"http://localhost:9090"}

echo "Testing Linux MCP Daemon Documentation Server..."

echo "Waiting for Documentation server to come online..."

MAX_RETRIES=10
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$DAEMON_URL/docs/")
    if [ "$HTTP_STATUS" -eq 200 ]; then
        echo "✅ SUCCESS: Documentation server returned HTTP 200 OK."
        exit 0
    fi
    
    echo "Server returned HTTP $HTTP_STATUS. Retrying in 2 seconds..."
    sleep 2
    RETRY_COUNT=$((RETRY_COUNT + 1))
done

echo "❌ FAILED: Documentation server never returned HTTP 200 after $MAX_RETRIES attempts."
exit 1
