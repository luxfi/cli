#!/usr/bin/env bash

set -e

# Find all shell scripts and run shellcheck on them
# Excludes:
#   - vendor directory
#   - .git directory
#   - pkg/ssh/shell/ - Go template files using {{ .Variable }} syntax
echo "Running shellcheck..."
find . -name "*.sh" -type f \
    -not -path "./vendor/*" \
    -not -path "./.git/*" \
    -not -path "./pkg/ssh/shell/*" \
    -exec shellcheck -S warning {} +

echo "Shellcheck passed!"