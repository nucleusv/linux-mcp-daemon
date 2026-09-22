#!/bin/bash
set -e

# Start daemon in background
export MCP_TOKEN="my-secret"
go build -o mcpd ./cmd/mcpd
./mcpd &
DAEMON_PID=$!

# Wait for it to start
sleep 2

# Connect to SSE, get the endpoint
echo "--- Connecting to SSE to get message endpoint ---"
ENDPOINT=$(curl -s -N -H "Authorization: Bearer my-secret" http://localhost:9090/sse | grep -m 1 "endpoint" | sed 's/.*endpoint=\([^"]*\).*/\1/')
echo "Got endpoint: $ENDPOINT"

# Fetch Tools
echo -e "\n--- Requesting tools/list ---"
curl -s -X POST "http://localhost:9090$ENDPOINT" \
     -H "Authorization: Bearer my-secret" \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}' | jq .

# Fetch Resources
echo -e "\n--- Requesting resources/list ---"
curl -s -X POST "http://localhost:9090$ENDPOINT" \
     -H "Authorization: Bearer my-secret" \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc": "2.0", "id": 2, "method": "resources/list"}' | jq .

# Cleanup
kill $DAEMON_PID
wait $DAEMON_PID 2>/dev/null || true
rm mcpd test_mcp_api.sh
