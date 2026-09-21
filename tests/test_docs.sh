#!/bin/bash
set -e

DAEMON_URL=${DAEMON_URL:-"http://localhost:9090"}

echo "Testing Linux MCP Daemon Documentation Server..."

HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$DAEMON_URL/docs/")

if [ "$HTTP_STATUS" -eq 200 ]; then
    echo "✅ SUCCESS: Documentation server returned HTTP 200 OK."
    exit 0
else
    echo "❌ FAILED: Documentation server returned HTTP $HTTP_STATUS"
    exit 1
fi
