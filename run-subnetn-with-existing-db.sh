#!/bin/bash
# Run SubnetEVM with existing 1M+ blocks database
# This uses the EVM plugin to load the existing blockchain

set -e

LUXD="/home/z/work/lux/node/build/luxd"
EVM_PLUGIN_ID="srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy"
EXISTING_DB="/home/z/work/lux/state/chaindata/lux-mainnet-96369"
DATA_DIR="/tmp/lux-subnet-existing"
CHAIN_ID="96369"

echo "=== LUX SubnetEVM with Existing 1M+ Blocks ==="
echo "Chain ID: $CHAIN_ID"
echo "Treasury: 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714"
echo "Database: $EXISTING_DB (7.2GB)"
echo "Expected blocks: 1,082,780+"
echo ""

# Kill any existing
pkill -9 luxd 2>/dev/null || true
sleep 2

# Setup directories
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"/{db,plugins,configs/chains}
mkdir -p "$DATA_DIR/staking"

# Copy EVM plugin
cp ~/.luxd/plugins/$EVM_PLUGIN_ID "$DATA_DIR/plugins/"
chmod +x "$DATA_DIR/plugins/$EVM_PLUGIN_ID"

# Link existing database for the EVM blockchain
# The blockchain ID for this subnet - we need to find it from configs
BLOCKCHAIN_ID=$(basename $(ls -d /home/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb 2>/dev/null | head -1) 2>/dev/null || echo "evm-chain")

# Create chain config pointing to existing DB
mkdir -p "$DATA_DIR/configs/chains/$EVM_PLUGIN_ID"
cat > "$DATA_DIR/configs/chains/$EVM_PLUGIN_ID/config.json" << EOF
{
  "snowman-api-enabled": false,
  "eth-apis": ["eth", "eth-filter", "net", "web3", "internal-eth", "internal-blockchain", "internal-transaction", "admin", "debug"],
  "rpc-gas-cap": 50000000,
  "rpc-tx-fee-cap": 100,
  "pruning-enabled": false,
  "log-level": "info",
  "continuous-profiler-enabled": false
}
EOF

# Copy original genesis
cp /home/z/work/lux/state/chaindata/configs/lux-mainnet-96369/genesis.json "$DATA_DIR/configs/chains/$EVM_PLUGIN_ID/genesis.json"

# Generate staking keys
openssl genrsa -out "$DATA_DIR/staking/staker.key" 4096 2>/dev/null
openssl req -new -x509 -key "$DATA_DIR/staking/staker.key" \
    -out "$DATA_DIR/staking/staker.crt" -days 365 \
    -subj "/C=US/ST=State/L=City/O=Lux/CN=luxnode" 2>/dev/null
cp "$DATA_DIR/staking/staker.key" "$DATA_DIR/staking/signer.key"

# Link the existing database
mkdir -p "$DATA_DIR/db/mainnet"
ln -sf "$EXISTING_DB/db/pebbledb" "$DATA_DIR/db/mainnet/pebbledb"

echo "Starting node..."

exec "$LUXD" \
    --dev \
    --network-id=$CHAIN_ID \
    --db-dir="$DATA_DIR/db" \
    --db-type=pebbledb \
    --plugin-dir="$DATA_DIR/plugins" \
    --chain-config-dir="$DATA_DIR/configs/chains" \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-port=9631 \
    --api-admin-enabled=true \
    --index-enabled=true \
    --log-level=info