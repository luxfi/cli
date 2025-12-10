#!/bin/bash
# Run C-Chain with migrated SubnetEVM database
set -e

LUXD="/home/z/work/lux/node/build/luxd"
MIGRATED_DB="/home/z/work/lux/state/chaindata/cchain-96369/db/pebbledb"
GENESIS="/home/z/work/lux/state/chaindata/lux-mainnet-96369/export/cchain-genesis.json"
DATA_DIR="/tmp/lux-cchain-migrated"
CHAIN_ID="96369"

echo "=== LUX C-Chain with Migrated Data ==="
echo "Chain ID: $CHAIN_ID"
echo "Database: $MIGRATED_DB"
echo "Expected blocks: 1,082,781"
echo ""

# Kill any existing
pkill -9 luxd 2>/dev/null || true
sleep 2

# Setup directories
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"/{staking,configs/chains/C}

# Copy genesis for C-Chain
cp "$GENESIS" "$DATA_DIR/configs/chains/C/genesis.json"

# C-Chain config
cat > "$DATA_DIR/configs/chains/C/config.json" << EOF
{
  "snowman-api-enabled": false,
  "eth-apis": ["eth", "eth-filter", "net", "web3", "internal-eth", "internal-blockchain", "internal-transaction", "admin", "debug", "personal", "txpool"],
  "rpc-gas-cap": 50000000,
  "rpc-tx-fee-cap": 100,
  "pruning-enabled": false,
  "log-level": "info",
  "state-sync-enabled": false,
  "local-txs-enabled": true
}
EOF

# Generate staking keys
openssl genrsa -out "$DATA_DIR/staking/staker.key" 4096 2>/dev/null
openssl req -new -x509 -key "$DATA_DIR/staking/staker.key" \
    -out "$DATA_DIR/staking/staker.crt" -days 365 \
    -subj "/C=US/ST=State/L=City/O=Lux/CN=luxnode" 2>/dev/null
cp "$DATA_DIR/staking/staker.key" "$DATA_DIR/staking/signer.key"

# Setup database - link migrated pebbledb for C-Chain
# C-Chain data goes to: db/<network-id>/C/
mkdir -p "$DATA_DIR/db/network-$CHAIN_ID/C"
ln -sf "$MIGRATED_DB" "$DATA_DIR/db/network-$CHAIN_ID/C/chaindata"

echo "Database structure:"
ls -la "$DATA_DIR/db/network-$CHAIN_ID/C/"
echo ""

echo "Starting luxd..."
echo "RPC: http://localhost:9630/ext/bc/C/rpc"

exec "$LUXD" \
    --dev \
    --network-id=$CHAIN_ID \
    --db-dir="$DATA_DIR/db" \
    --db-type=pebbledb \
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
