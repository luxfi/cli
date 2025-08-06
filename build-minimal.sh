#!/bin/bash
# Minimal build script that excludes problematic packages

echo "Building minimal CLI without SDK and problematic packages..."

# Build with tags to exclude SDK
go build -tags="nosdk" \
  -ldflags "-X 'github.com/luxfi/cli/cmd.Version=v1.9.2-lux'" \
  -o bin/lux \
  main.go

if [ $? -eq 0 ]; then
  echo "Build successful! Binary at bin/lux"
  ./bin/lux --version
else
  echo "Build failed, trying even more minimal build..."
  
  # Try building with just the core commands
  go build \
    -ldflags "-X 'github.com/luxfi/cli/cmd.Version=v1.9.2-lux-minimal'" \
    -o bin/lux-minimal \
    main.go
fi