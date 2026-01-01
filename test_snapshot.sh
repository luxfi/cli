#!/bin/bash
# Test script for snapshot functionality

set -e

echo "Testing Lux CLI snapshot commands..."
echo ""

# Check if lux binary exists
if [ ! -f "./bin/lux" ]; then
    echo "Error: lux binary not found at ./bin/lux"
    echo "Please build the binary first with: make build"
    exit 1
fi

# Test help for snapshot command
echo "1. Testing 'lux network snapshot --help'"
./bin/lux network snapshot --help
echo "✓ Snapshot help works"
echo ""

# Test save command help
echo "2. Testing 'lux network snapshot save --help'"
./bin/lux network snapshot save --help
echo "✓ Snapshot save help works"
echo ""

# Test list command help
echo "3. Testing 'lux network snapshot list --help'"
./bin/lux network snapshot list --help
echo "✓ Snapshot list help works"
echo ""

# Test load command help
echo "4. Testing 'lux network snapshot load --help'"
./bin/lux network snapshot load --help
echo "✓ Snapshot load help works"
echo ""

# Test delete command help
echo "5. Testing 'lux network snapshot delete --help'"
./bin/lux network snapshot delete --help
echo "✓ Snapshot delete help works"
echo ""

echo "All snapshot command tests passed! ✓"
