#!/bin/bash
set -e

# Change to the root directory of the project
cd "$(dirname "$0")/.."

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}===========================================${NC}"
echo -e "${BLUE}   Linux MCP Daemon - Full Deployment      ${NC}"
echo -e "${BLUE}===========================================${NC}\n"

echo -e "📦 ${GREEN}Step 1: Building daemon container image...${NC}"
./scripts/build.sh
echo ""

echo -e "📦 ${GREEN}Step 2: Building linuxctl CLI locally...${NC}"
if [ -x "./scripts/build-cli.sh" ]; then
    ./scripts/build-cli.sh
else
    bash ./scripts/build-cli.sh
fi
echo ""

echo -e "🚀 ${GREEN}Step 3: Deploying to Kubernetes...${NC}"
./scripts/deploy.sh
echo ""

echo -e "🧪 ${GREEN}Step 4: Running integration test suite...${NC}"
./tests/run_all.sh
echo ""

echo -e "${BLUE}===========================================${NC}"
echo -e "${GREEN}   Deployment & Testing Complete! 🎉        ${NC}"
echo -e "${BLUE}===========================================${NC}"
