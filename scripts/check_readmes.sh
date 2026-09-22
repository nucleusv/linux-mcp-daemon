#!/bin/bash

# This script checks that every terminal directory inside internal/tools/ and internal/resources/
# contains a README.md file, enforcing documentation standards for AI agents.

set -e

EXIT_CODE=0

echo "Checking for missing README.md files..."

# Find all directories that contain go files (which means it's a leaf package)
for dir in $(find internal/tools internal/resources -type f -name '*.go' -exec dirname {} \; | sort -u); do
    # Skip directories that are not terminal/feature packages (e.g. internal/tools/get_os_release might be old, but let's check anyway)
    if [ ! -f "$dir/README.md" ]; then
        echo "❌ Missing README.md in: $dir"
        EXIT_CODE=1
    fi
done

if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ All packages have a README.md!"
else
    echo ""
    echo "Please add a README.md to the above directories explaining the tool/resource."
fi

exit $EXIT_CODE
