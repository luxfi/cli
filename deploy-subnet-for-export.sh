#!/bin/bash

# Deploy SubnetEVM with existing data for export
echo "Deploying SubnetEVM with existing blockchain data..."

# Configuration
SUBNET_ID="2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB"
VM_ID="srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy"
PORT=9640
DATA_DIR="/home/z/.luxd-subnet-export"
PLUGIN_DIR="$DATA_DIR/plugins"

# Clean up old deployment
pkill -f "port=$PORT" 2>/dev/null || true
rm -rf $DATA_DIR

# Create directory structure
mkdir -p $DATA_DIR/{staking,plugins,logs,db,configs/chains}
mkdir -p $DATA_DIR/configs/chains/$SUBNET_ID

# Generate staking certificates
cd $DATA_DIR/staking
openssl req -x509 -newkey rsa:4096 -keyout staker.key -out staker.crt -sha256 -days 365 -nodes -subj "/CN=SubnetNode" 2>/dev/null
cp staker.key signer.key

# Find and link existing SubnetEVM plugin
PLUGIN_SOURCE="/home/z/work/lux/evm/build/evm"
if [ ! -f "$PLUGIN_SOURCE" ]; then
    PLUGIN_SOURCE="/home/z/.luxd-5node-rpc/node2/plugins/$VM_ID"
fi
if [ ! -f "$PLUGIN_SOURCE" ]; then
    echo "Building SubnetEVM plugin..."
    cd /home/z/work/lux/evm
    ./scripts/build.sh
    PLUGIN_SOURCE="/home/z/work/lux/evm/build/evm"
fi

if [ -f "$PLUGIN_SOURCE" ]; then
    cp "$PLUGIN_SOURCE" "$PLUGIN_DIR/$VM_ID"
    chmod +x "$PLUGIN_DIR/$VM_ID"
    echo "Plugin installed: $PLUGIN_DIR/$VM_ID"
else
    echo "Warning: SubnetEVM plugin not found"
fi

# Link existing blockchain data (read-only)
EXISTING_DB="/home/z/.luxd-5node-rpc/node2/chains/$SUBNET_ID/db"
if [ ! -d "$EXISTING_DB" ]; then
    EXISTING_DB="/home/z/.lux-cli/runs/mainnet-regenesis/node1/chains/$SUBNET_ID/db"
fi
if [ -d "$EXISTING_DB" ]; then
    mkdir -p $DATA_DIR/chains/$SUBNET_ID
    ln -s "$EXISTING_DB" "$DATA_DIR/chains/$SUBNET_ID/db"
    echo "Linked existing database: $EXISTING_DB"
else
    echo "Warning: No existing SubnetEVM database found"
fi

# Create chain config
cat > $DATA_DIR/configs/chains/$SUBNET_ID/config.json <<EOF
{
  "chain-id": 96369,
  "homestead-block": 0,
  "eip150-block": 0,
  "eip150-hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "eip155-block": 0,
  "eip158-block": 0,
  "byzantium-block": 0,
  "constantinople-block": 0,
  "petersburg-block": 0,
  "istanbul-block": 0,
  "muir-glacier-block": 0,
  "subnet-evm-timestamp": 0,
  "fee-config": {
    "gas-limit": 15000000,
    "min-base-fee": 100000000000,
    "target-gas": 15000000,
    "base-fee-change-denominator": 36,
    "min-block-gas-cost": 0,
    "max-block-gas-cost": 1000000,
    "target-block-rate": 2,
    "block-gas-cost-step": 200000
  },
  "allow-fee-recipients": false
}
EOF

# Create node config
cat > $DATA_DIR/config.json <<EOF
{
  "network-id": 96369,
  "data-dir": "$DATA_DIR",
  "db-dir": "$DATA_DIR/db",
  "log-dir": "$DATA_DIR/logs",
  "plugin-dir": "$PLUGIN_DIR",
  "chain-config-dir": "$DATA_DIR/configs/chains",
  "log-level": "info",
  "http-host": "0.0.0.0",
  "http-port": $PORT,
  "staking-enabled": false,
  "sybil-protection-enabled": false,
  "consensus-sample-size": 1,
  "consensus-quorum-size": 1,
  "api-admin-enabled": true,
  "index-enabled": true,
  "db-type": "pebbledb",
  "http-allowed-origins": "*",
  "http-allowed-hosts": "*",
  "track-subnets": "$SUBNET_ID"
}
EOF

# Start the node
echo "Starting SubnetEVM node on port $PORT..."
/home/z/work/lux/node/build/luxd --config-file=$DATA_DIR/config.json &

echo "Waiting for SubnetEVM to be ready..."
sleep 10

# Test the endpoint
echo "Testing SubnetEVM RPC endpoint..."
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:$PORT/ext/bc/$SUBNET_ID/rpc

echo ""
echo "SubnetEVM deployed!"
echo "RPC endpoint: http://localhost:$PORT/ext/bc/$SUBNET_ID/rpc"