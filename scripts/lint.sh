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

# Run golangci-lint if available
if command -v golangci-lint &> /dev/null; then
    echo "Running golangci-lint..."
    golangci-lint run
else
    echo "golangci-lint not found, skipping..."
fi

echo "All checks passed!"