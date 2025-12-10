#!/bin/bash

# Minimal launch script for C-Chain in dev mode
# Uses only essential flags to get the node running

echo "Starting LUX node with minimal configuration..."

# Kill any existing luxd process
pkill -f luxd 2>/dev/null

# Wait a moment for port to be released
sleep 2

# Start luxd with minimal flags
/home/z/work/lux/node/build/luxd \
  --network-id=96369 \
  --http-port=9630 \
  --data-dir=/home/z/.luxd \
  --chain-data-dir=/home/z/.luxd/chainData \
  --dev \
  --skip-bootstrap \
  --log-level=info 2>&1 | tee /home/z/work/lux/cli/luxd-minimal.log