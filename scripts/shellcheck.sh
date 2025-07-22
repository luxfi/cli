#!/usr/bin/env bash

set -e

# Find all shell scripts and run shellcheck on them
echo "Running shellcheck..."
find . -name "*.sh" -type f -not -path "./vendor/*" -not -path "./.git/*" | xargs shellcheck

echo "Shellcheck passed!"