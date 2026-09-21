#!/bin/bash
# scripts/build-cli.sh
set -e

# Change to the root directory of the project
cd "$(dirname "$0")/.."

echo "Building linuxctl for macOS..."

# Build for the local machine (assumes macOS/darwin by default if run on Mac)
go build -o linuxctl ./cmd/linuxctl

echo "Build complete. You can run it via ./linuxctl"
