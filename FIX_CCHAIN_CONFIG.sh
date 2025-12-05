#!/bin/bash

# Fix C-Chain configuration for luxd with migrated database

echo "=== Fixing C-Chain Configuration ==="
echo

WORK_DIR="/home/z/.luxd-mainnet"
LUXD_BIN="/home/z/work/lux/node/build/luxd"

# Stop any running luxd
echo "Stopping any running luxd..."
pkill -f luxd || true
sleep 2

# Remove incorrect chains.json file
echo "Removing incorrect chains.json configuration..."
rm -f "$WORK_DIR/configs/chains.json"

# The C-Chain doesn't need a chains.json entry - it's a primary network chain
# It will be created automatically with the correct chain ID

# Create a proper chain config for C-Chain (if needed)
mkdir -p "$WORK_DIR/configs/chains/C"
cat > "$WORK_DIR/configs/chains/C/config.json" << 'JSON'
{
  "snowman-api-enabled": false,
  "coreth-admin-api-enabled": false,
  "eth-apis": ["internal-public-eth", "internal-public-debug", "internal-public-account", "internal-public-personal", "internal-debug-handler", "internal-public-health", "internal-public-net", "internal-public-txpool", "internal-public-web3"],
  "rpc-gas-cap": 50000000,
  "rpc-tx-fee-cap": 100,
  "pruning-enabled": false,
  "health-check-frequency": "30s",
  "max-block-history-lookback": 0,
  "log-level": "info"
}
JSON

echo
echo "=== Starting luxd with corrected configuration ==="
echo
echo "Data dir: $WORK_DIR"
echo "Using genesis data with migrated C-Chain database"
echo

# Start luxd with proper configuration
"$LUXD_BIN" \
    --network-id=96369 \
    --data-dir="$WORK_DIR" \
    --genesis-file=/home/z/work/lux/genesis-mainnet/genesis.json \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-port=9631 \
    --db-dir="$WORK_DIR/db" \
    --chain-data-dir="$WORK_DIR/chainData" \
    --log-level=info \
    --log-dir="$WORK_DIR/logs" \
    --public-ip=127.0.0.1 \
    --dev \
    --health-check-frequency=30s \
    --index-enabled=true \
    --api-admin-enabled=true \
    --api-ipcs-enabled=true \
    --api-keystore-enabled=false \
    --api-metrics-enabled=true \
    --chain-config-dir="$WORK_DIR/configs/chains" \
    --http-allowed-origins="*" \
    --http-allowed-hosts="*" > /tmp/luxd_mainnet.log 2>&1 &

echo "luxd started with PID $!"
echo "Logs: /tmp/luxd_mainnet.log"
echo
echo "Waiting for node to initialize..."
sleep 10

echo
echo "=== Testing C-Chain RPC ==="

# Test health
echo "Testing node health..."
curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"health.health"}' \
  -H 'content-type:application/json' http://127.0.0.1:9630/ext/health | jq .

# Get blockchain ID for C-Chain
echo
echo "Getting C-Chain blockchain ID..."
CHAIN_ID=$(curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"info.getBlockchainID","params":{"alias":"C"}}' \
  -H 'content-type:application/json' http://127.0.0.1:9630/ext/info | jq -r .result.blockchainID)

if [ "$CHAIN_ID" = "null" ] || [ -z "$CHAIN_ID" ]; then
    echo "ERROR: Could not get C-Chain ID"
    echo "Checking log for errors:"
    tail -50 /tmp/luxd_mainnet.log
    exit 1
fi

echo "C-Chain blockchain ID: $CHAIN_ID"

# Test C-Chain RPC
echo
echo "Testing C-Chain RPC..."
echo "URL: http://127.0.0.1:9630/ext/bc/$CHAIN_ID/rpc"

# Get block number
echo
echo "Getting block number..."
curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  -H 'content-type:application/json' "http://127.0.0.1:9630/ext/bc/$CHAIN_ID/rpc" | jq .

# Check luxdefi.eth balance
echo
echo "Checking luxdefi.eth balance..."
curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x9011E888251AB053B7bD1cdB598Db4f9DEd94714","latest"],"id":1}' \
  -H 'content-type:application/json' "http://127.0.0.1:9630/ext/bc/$CHAIN_ID/rpc" | jq .

echo
echo "=== Configuration Fixed ==="
echo "C-Chain is now properly configured and running"
echo "RPC endpoint: http://127.0.0.1:9630/ext/bc/$CHAIN_ID/rpc"
echo "WebSocket: ws://127.0.0.1:9630/ext/bc/$CHAIN_ID/ws"