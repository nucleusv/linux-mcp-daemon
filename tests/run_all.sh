#!/bin/bash
set -e

# Change to the tests directory
cd "$(dirname "$0")"

# Parse arguments
MANUAL_REVIEW="no"
for arg in "$@"; do
    if [[ "$arg" == "manual_review=yes" ]]; then
        MANUAL_REVIEW="yes"
    fi
done

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}===========================================${NC}"
echo -e "${BLUE}   Linux MCP Daemon Integration Tests      ${NC}"
echo -e "${BLUE}===========================================${NC}\n"

run_test() {
    local script=$1
    local name=$2

    echo -e "⏳ Running: ${name}..."
    if ./"$script"; then
        echo -e "${GREEN}✓ Passed: ${name}${NC}\n"
    else
        echo -e "${RED}✗ Failed: ${name}${NC}\n"
        echo -e "${RED}Test suite aborted due to failure.${NC}"
        exit 1
    fi

    if [[ "$MANUAL_REVIEW" == "yes" ]]; then
        echo -e "${BLUE}Manual review mode: Press Enter to continue to the next test...${NC}"
        read -r
    fi
}

# Run tests in logical order
run_test "test_docs.sh" "Documentation Server Test"
run_test "test_mcp.sh" "MCP SSE Protocol End-to-End Test"
run_test "test_linuxctl.sh" "Linuxctl CLI Ping Test"

echo -e "${GREEN}===========================================${NC}"
echo -e "${GREEN}   All tests completed successfully! 🎉      ${NC}"
echo -e "${GREEN}===========================================${NC}"
