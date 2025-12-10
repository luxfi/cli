#!/bin/bash
# Run SubnetEVM node in read-only mode to query state via RPC
set -e

LUXD="/home/z/work/lux/node/build/luxd"
VM_ID="srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy"
BLOCKCHAIN_ID="dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ"
EXISTING_DB="/home/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb"
CONFIG_DIR="/home/z/work/lux/state/chaindata/configs/lux-mainnet-96369"
DATA_DIR="/tmp/lux-readonly-subnet"
CHAIN_ID="96369"

echo "=== LUX SubnetEVM Read-Only Mode ==="
echo "Chain ID: $CHAIN_ID"
echo "Blockchain ID: $BLOCKCHAIN_ID"
echo ""

# Kill any existing
pkill -9 luxd 2>/dev/null || true
sleep 2

# Setup directories - fresh each time
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"/{plugins,staking}
mkdir -p "$DATA_DIR/configs/chains/$BLOCKCHAIN_ID"

# For SubnetEVM, DB is at: db/<network>/<blockchain_id>/chaindata
# The namespace 337fb73f... maps to the blockchain data inside pebbledb
mkdir -p "$DATA_DIR/db/mainnet/$BLOCKCHAIN_ID"

# Copy the VM plugin
if [ -f ~/.luxd/plugins/$VM_ID ]; then
    cp ~/.luxd/plugins/$VM_ID "$DATA_DIR/plugins/"
elif [ -f /home/z/work/lux/evm/build/$VM_ID ]; then
    cp /home/z/work/lux/evm/build/$VM_ID "$DATA_DIR/plugins/"
fi
chmod +x "$DATA_DIR/plugins/$VM_ID" 2>/dev/null || true

# Chain config
cat > "$DATA_DIR/configs/chains/$BLOCKCHAIN_ID/config.json" << EOF
{
  "snowman-api-enabled": false,
  "eth-apis": ["eth", "eth-filter", "net", "web3", "internal-eth", "internal-blockchain", "internal-transaction", "admin", "debug"],
  "rpc-gas-cap": 50000000,
  "rpc-tx-fee-cap": 100,
  "pruning-enabled": false,
  "log-level": "debug",
  "state-sync-enabled": false
}
EOF

# Use original genesis
cp "$CONFIG_DIR/genesis.original.json" "$DATA_DIR/configs/chains/$BLOCKCHAIN_ID/genesis.json"

# Generate staking keys
openssl genrsa -out "$DATA_DIR/staking/staker.key" 4096 2>/dev/null
openssl req -new -x509 -key "$DATA_DIR/staking/staker.key" \
    -out "$DATA_DIR/staking/staker.crt" -days 365 \
    -subj "/C=US/ST=State/L=City/O=Lux/CN=luxnode" 2>/dev/null
cp "$DATA_DIR/staking/staker.key" "$DATA_DIR/staking/signer.key"

# Create symlink to the existing pebbledb
# SubnetEVM expects data at db/<network_id>/<chain_id>
ln -sf "$EXISTING_DB" "$DATA_DIR/db/mainnet/$BLOCKCHAIN_ID/pebbledb"

echo "Starting luxd in dev mode..."
echo "RPC will be at: http://localhost:9630/ext/bc/$BLOCKCHAIN_ID/rpc"

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
    --log-level=info
