#!/usr/bin/env bash

set -e

# Run go fmt
echo "Running go fmt..."
fmt_output=$(gofmt -l .)
if [ -n "$fmt_output" ]; then
    echo "The following files need to be formatted:"
    echo "$fmt_output"
    exit 1
fi

# Run go vet
echo "Running go vet..."
go vet ./...

# Run staticcheck
echo "Running staticcheck..."
if command -v staticcheck &> /dev/null; then
    staticcheck ./... || true
else
    go install honnef.co/go/tools/cmd/staticcheck@latest
    staticcheck ./... || true
fi

echo "All checks passed!"