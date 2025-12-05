#!/bin/bash
# Run SubnetEVM node with original blockchain ID to load 1M+ blocks
# Uses the exact blockchain ID from the original deployment

set -e

LUXD="/home/z/work/lux/node/build/luxd"
BLOCKCHAIN_ID="dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ"
SUBNET_ID="tJqmx13PV8UPQJBbuumANQCKnfPUHCxfahdG29nJa6BHkumCK"
VM_ID="srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy"
EXISTING_DB="/home/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb"
CONFIG_DIR="/home/z/work/lux/state/chaindata/configs/lux-mainnet-96369"
DATA_DIR="/tmp/lux-original-subnet"
CHAIN_ID="96369"

echo "=== LUX SubnetEVM with Original Blockchain ==="
echo "Chain ID: $CHAIN_ID"
echo "Blockchain ID: $BLOCKCHAIN_ID"
echo "Subnet ID: $SUBNET_ID"
echo "VM ID: $VM_ID"
echo "Database: $EXISTING_DB (7.2GB)"
echo "Expected blocks: 1,082,780+"
echo ""

# Kill any existing
pkill -9 luxd 2>/dev/null || true
sleep 2

# Setup directories
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"/{plugins,staking}
mkdir -p "$DATA_DIR/configs/chains/$BLOCKCHAIN_ID"
mkdir -p "$DATA_DIR/db/network-$CHAIN_ID"

# Copy EVM plugin with correct VM ID
cp ~/.luxd/plugins/$VM_ID "$DATA_DIR/plugins/" 2>/dev/null || \
    cp /home/z/work/lux/evm/build/$VM_ID "$DATA_DIR/plugins/" 2>/dev/null || \
    echo "Warning: Could not copy VM plugin"
chmod +x "$DATA_DIR/plugins/$VM_ID" 2>/dev/null || true

# Chain config for the specific blockchain
cat > "$DATA_DIR/configs/chains/$BLOCKCHAIN_ID/config.json" << EOF
{
  "snowman-api-enabled": false,
  "eth-apis": ["eth", "eth-filter", "net", "web3", "internal-eth", "internal-blockchain", "internal-transaction", "admin", "debug"],
  "rpc-gas-cap": 50000000,
  "rpc-tx-fee-cap": 100,
  "pruning-enabled": false,
  "log-level": "info",
  "state-sync-enabled": false
}
EOF

# Copy original genesis
cp "$CONFIG_DIR/genesis.original.json" "$DATA_DIR/configs/chains/$BLOCKCHAIN_ID/genesis.json"

# Generate staking keys
openssl genrsa -out "$DATA_DIR/staking/staker.key" 4096 2>/dev/null
openssl req -new -x509 -key "$DATA_DIR/staking/staker.key" \
    -out "$DATA_DIR/staking/staker.crt" -days 365 \
    -subj "/C=US/ST=State/L=City/O=Lux/CN=luxnode" 2>/dev/null
cp "$DATA_DIR/staking/staker.key" "$DATA_DIR/staking/signer.key"

# Link the existing database under the blockchain ID directory
# SubnetEVM stores data at: db/network-{id}/{blockchain_id}/
mkdir -p "$DATA_DIR/db/network-$CHAIN_ID/$BLOCKCHAIN_ID"
ln -sf "$EXISTING_DB" "$DATA_DIR/db/network-$CHAIN_ID/$BLOCKCHAIN_ID/pebbledb"

# Also link to common locations
mkdir -p "$DATA_DIR/db/network-$CHAIN_ID/chains"
ln -sf "$EXISTING_DB" "$DATA_DIR/db/network-$CHAIN_ID/chains/$BLOCKCHAIN_ID"

echo "Database structure:"
ls -la "$DATA_DIR/db/network-$CHAIN_ID/"
echo ""

echo "Starting node..."

exec "$LUXD" \
    --dev \
    --network-id=$CHAIN_ID \
    --db-dir="$DATA_DIR/db" \
    --db-type=pebbledb \
    --plugin-dir="$DATA_DIR/plugins" \
    --chain-config-dir="$DATA_DIR/configs/chains" \
    --staking-tls-cert-file="$DATA_DIR/staking/staker.crt" \
    --staking-tls-key-file="$DATA_DIR/staking/staker.key" \
    --staking-signer-key-file="$DATA_DIR/staking/signer.key" \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-port=9631 \
    --api-admin-enabled=true \
    --index-enabled=true \
    --log-level=debug
