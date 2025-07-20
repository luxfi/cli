#!/bin/bash
# Build script to test validator CLI integration

echo "Building Lux CLI with validator commands..."

# Build just the CLI binary
cd /home/z/work/lux/cli
go build -tags netgo -ldflags="-s -w" -o bin/lux-test ./main.go

if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo ""
    echo "Testing validator commands:"
    ./bin/lux-test node validator --help
else
    echo "Build failed"
    exit 1
fi