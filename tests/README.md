# Linux MCP Daemon Test Suite

This directory contains end-to-end integration test scripts that verify the `mcpd` is functioning properly over HTTP/HTTPS.

## Prerequisites

Ensure the daemon is running locally or deployed.

## Tests

### `test_docs.sh`
Verifies that the Docusaurus statically compiled documentation is being successfully served on the `/docs/` route by the master daemon.

**Usage:**
```bash
./test_docs.sh
```

### `test_mcp.sh`
Verifies the complete end-to-end MCP workflow. This script:
1. Connects to the daemon's `/sse` stream.
2. Intercepts the `endpoint` URL.
3. Sends a JSON-RPC POST request to `/message` using `get_sudo_rules`.
4. Asserts that the daemon successfully parsed the JSON, spawned a worker, elevated privileges (if applicable), and streamed the result back over the active SSE connection.

**Usage:**
```bash
./test_mcp.sh
```

You can optionally override environment variables:
```bash
DAEMON_URL="http://localhost:9090" TOKEN="my-test-token-123" ./test_mcp.sh
```
