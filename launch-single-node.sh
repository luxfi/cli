#!/bin/bash
# Single-node C-Chain launcher with bootstrap bypass
# Forces C-Chain initialization without waiting for peers

LUXD="/home/z/work/lux/node/build/luxd"
DATA_DIR="/home/z/.luxd"

# Kill any existing instances
pkill -f luxd 2>/dev/null
sleep 2

echo "Starting single-node LUX with C-Chain forced initialization..."

# Launch with dev mode and POA single-node flags
$LUXD \
  --data-dir="$DATA_DIR" \
  --network-id=96369 \
  --http-port=9630 \
  --http-host=0.0.0.0 \
  --dev \
  --skip-bootstrap=true \
  --poa-single-node-mode=true \
  --poa-mode-enabled=true \
  --enable-automining=true \
  --bootstrap-ips="" \
  --bootstrap-ids="" \
  --chain-data-dir="$DATA_DIR/chainData" \
  --db-type=badgerdb \
  --c-chain-db-type=badgerdb \
  --log-level=debug \
  --log-display-level=info