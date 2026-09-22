#!/bin/bash
# scripts/build.sh
set -e

# Change to the root directory of the project
cd "$(dirname "$0")/.."

echo "Building Docker image 'linux-mcp-daemon-by-claude:local'..."
docker build -t linux-mcp-daemon-by-claude:local .
echo "Build complete."
