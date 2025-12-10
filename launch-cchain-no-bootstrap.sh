#!/bin/bash

# Launch script for C-Chain with skip-bootstrap to bypass peer requirements
# This enables single-node POA mode with the migrated database

echo "Starting LUX node with skip-bootstrap for C-Chain..."

# Kill any existing luxd process
pkill -f luxd 2>/dev/null

# Wait a moment for port to be released
sleep 2

# Start luxd with skip-bootstrap flag
/home/z/work/lux/node/build/luxd \
  --network-id=96369 \
  --http-port=9630 \
  --staking-port=9631 \
  --data-dir=/home/z/.luxd \
  --db-dir=/home/z/.luxd/db \
  --sybil-protection-enabled=false \
  --snow-sample-size=1 \
  --snow-quorum-size=1 \
  --snow-virtuous-commit-threshold=1 \
  --snow-rogue-commit-threshold=1 \
  --api-admin-enabled=true \
  --api-eth-enabled=true \
  --coreth-admin-api-enabled=true \
  --coreth-continuous-profiler-dir="" \
  --coreth-continuous-profiler-frequency=0 \
  --coreth-offline-pruning-enabled=false \
  --public-ip=127.0.0.1 \
  --min-stake-duration=0s \
  --staking-tls-cert-file=/home/z/.luxd/staking/local/staker.crt \
  --staking-tls-key-file=/home/z/.luxd/staking/local/staker.key \
  --http-allowed-origins="*" \
  --http-allowed-hosts="*" \
  --chain-aliases-file=/home/z/.luxd/configs/chains/aliases.json \
  --chain-data-dir=/home/z/.luxd/chainData \
  --skip-bootstrap \
  --log-level=info 2>&1 | tee /home/z/work/lux/cli/luxd-cchain.log