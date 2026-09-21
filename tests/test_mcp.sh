#!/bin/bash
set -e

DAEMON_URL=${DAEMON_URL:-"http://localhost:9090"}
TOKEN=${TOKEN:-"my-test-token-123"}
LOG_FILE="sse.log"

echo "Testing Linux MCP Daemon SSE and Tool Execution..."

# 1. Start the SSE connection in the background
> $LOG_FILE
curl -N -s -H "Authorization: Bearer $TOKEN" "$DAEMON_URL/sse" > $LOG_FILE &
SSE_PID=$!

# Ensure we cleanup the background process on exit
cleanup() {
    kill $SSE_PID 2>/dev/null || true
    rm -f $LOG_FILE
}
trap cleanup EXIT

# 2. Wait for connection
echo "Waiting for SSE endpoint initialization..."
sleep 5

if ! grep -q "event: endpoint" $LOG_FILE; then
    echo "❌ FAILED: Did not receive endpoint event over SSE."
    cat $LOG_FILE
    exit 1
fi

echo "✅ SSE connected successfully."

# 3. Fire an asynchronous tool call (get_sudo_rules)
echo "Sending JSON-RPC 'get_sudo_rules' execution request to /message..."

PAYLOAD='{
  "jsonrpc": "2.0",
  "id": 999,
  "method": "tools/call",
  "params": {
    "name": "get_sudo_rules",
    "arguments": {}
  }
}'

ENDPOINT=$(grep "data: /message" $LOG_FILE | cut -d' ' -f2 | tr -d '\r')
if [ -z "$ENDPOINT" ]; then
    echo "❌ FAILED: Could not extract dynamic SSE POST endpoint."
    cat $LOG_FILE
    exit 1
fi
echo "Extracted endpoint: $ENDPOINT"

curl -v -s -X POST "$DAEMON_URL$ENDPOINT" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "$PAYLOAD" || echo "❌ curl POST failed!"

# 4. Wait for the daemon to process the worker and stream the response
echo "Waiting for JSON-RPC response..."
sleep 5

if grep -q "get_list_of_files" $LOG_FILE; then
    echo "✅ SUCCESS: Daemon successfully executed the tool and streamed the result back!"
    exit 0
else
    echo "❌ FAILED: Did not find the expected tool output in the SSE stream."
    echo "SSE Stream Content:"
    cat $LOG_FILE
    exit 1
fi
