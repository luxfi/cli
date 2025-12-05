#!/bin/bash

# Simple C-Chain startup script

echo "=== Starting C-Chain with Migrated Database ==="
echo

WORK_DIR="/home/z/.luxd-mainnet"
LUXD_BIN="/home/z/work/lux/node/build/luxd"

# Stop any running luxd
echo "Stopping any running luxd..."
pkill -f luxd || true
sleep 2

# Clean up incorrect configuration
echo "Cleaning up configuration..."
rm -f "$WORK_DIR/configs/chains.json"

echo
echo "Starting luxd in dev mode..."
echo "Data directory: $WORK_DIR"
echo

# Start luxd with minimal configuration in dev mode
"$LUXD_BIN" \
    --network-id=96369 \
    --data-dir="$WORK_DIR" \
    --genesis-file=/home/z/work/lux/node/genesis/genesis_96369_migrated.json \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-port=9631 \
    --chain-data-dir="$WORK_DIR/chainData" \
    --log-level=info \
    --dev \
    --http-allowed-origins="*" \
    --http-allowed-hosts="*" > /tmp/luxd_mainnet.log 2>&1 &

PID=$!
echo "luxd started with PID $PID"
echo "Logs: tail -f /tmp/luxd_mainnet.log"
echo

# Wait for initialization
echo "Waiting for node to initialize..."
for i in {1..30}; do
    if curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"health.health"}' \
         -H 'content-type:application/json' http://127.0.0.1:9630/ext/health 2>/dev/null | grep -q healthy; then
        echo "Node is healthy!"
        break
    fi
    sleep 1
    if [ $i -eq 30 ]; then
        echo "Node failed to become healthy. Check logs:"
        tail -50 /tmp/luxd_mainnet.log
        exit 1
    fi
done

echo
echo "=== Testing C-Chain ==="

# Get C-Chain blockchain ID
echo "Getting C-Chain blockchain ID..."
RESPONSE=$(curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"info.getBlockchainID","params":{"alias":"C"}}' \
  -H 'content-type:application/json' http://127.0.0.1:9630/ext/info)

CHAIN_ID=$(echo "$RESPONSE" | jq -r .result.blockchainID 2>/dev/null)

if [ "$CHAIN_ID" = "null" ] || [ -z "$CHAIN_ID" ]; then
    echo "ERROR: Could not get C-Chain ID"
    echo "Response: $RESPONSE"
    echo
    echo "Checking logs:"
    tail -50 /tmp/luxd_mainnet.log | grep -E "ERROR|WARN|chain|C-Chain|evm"
    exit 1
fi

echo "C-Chain blockchain ID: $CHAIN_ID"
echo

# Test eth_blockNumber
echo "Testing eth_blockNumber..."
curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  -H 'content-type:application/json' "http://127.0.0.1:9630/ext/bc/$CHAIN_ID/rpc" | jq .

echo
echo "Testing balance of luxdefi.eth (0x9011E888251AB053B7bD1cdB598Db4f9DEd94714)..."
curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x9011E888251AB053B7bD1cdB598Db4f9DEd94714","latest"],"id":1}' \
  -H 'content-type:application/json' "http://127.0.0.1:9630/ext/bc/$CHAIN_ID/rpc" | jq .

echo
echo "=== C-Chain is running ==="
echo "RPC endpoint: http://127.0.0.1:9630/ext/bc/$CHAIN_ID/rpc"
echo "WebSocket: ws://127.0.0.1:9630/ext/bc/$CHAIN_ID/ws"
echo
echo "To check logs: tail -f /tmp/luxd_mainnet.log"
echo "To stop: kill $PID"