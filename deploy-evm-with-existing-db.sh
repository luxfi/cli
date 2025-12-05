#!/bin/bash
# Deploy EVM blockchain with existing SubnetEVM database
# This properly loads the 850,870 blocks from the PebbleDB database

set -e

# Configuration
EXISTING_DB="/home/z/work/lux/state/chaindata/lux-mainnet-96369"
LUXD="/home/z/work/lux/node/build/luxd"
EVM_BINARY="/home/z/work/lux/evm/build/evm"
DATA_DIR="/tmp/lux-evm-migration"
CHAIN_ID=96369
TREASURY="0x9011E888251AB053B7bD1cdB598Db4f9DEd94714"

echo "=== LUX EVM Migration with Existing Database ==="
echo "Chain ID: $CHAIN_ID"
echo "Treasury: $TREASURY (2T LUX)"
echo "Existing DB: $EXISTING_DB"
echo "Expected blocks: 850,870"
echo ""

# Check database exists
if [ ! -d "$EXISTING_DB/db/pebbledb" ]; then
    echo "ERROR: Database not found at $EXISTING_DB/db/pebbledb"
    exit 1
fi

# Check database size
DB_SIZE=$(du -sh "$EXISTING_DB/db/pebbledb" | cut -f1)
echo "Database size: $DB_SIZE (expected ~7.2GB)"
echo ""

# Kill any existing processes
echo "Cleaning up existing processes..."
pkill -9 luxd 2>/dev/null || true
pkill -9 evm 2>/dev/null || true
sleep 2

# Setup directories
echo "Setting up directories..."
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"/{db,plugins,configs/chains/C}
mkdir -p "$DATA_DIR"/staking

# Copy EVM binary as plugin
echo "Installing EVM plugin..."
if [ -f "$EVM_BINARY" ]; then
    cp "$EVM_BINARY" "$DATA_DIR/plugins/srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy"
    chmod +x "$DATA_DIR/plugins/srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy"
else
    echo "WARNING: EVM binary not found at $EVM_BINARY"
fi

# Create genesis with correct treasury
cat > "$DATA_DIR/configs/chains/C/config.json" << EOF
{
  "snowman-api-enabled": false,
  "coreth-admin-api-enabled": true,
  "eth-apis": ["eth", "eth-filter", "net", "web3", "internal-eth", "internal-blockchain", "internal-transaction", "admin", "debug", "personal", "txpool"],
  "rpc-gas-cap": 50000000,
  "rpc-tx-fee-cap": 100,
  "pruning-enabled": false,
  "log-level": "info",
  "db-type": "pebble",
  "import-db-path": "$EXISTING_DB/db/pebbledb",
  "continuous-profiler-enabled": false
}
EOF

# Create genesis configuration
cat > "$DATA_DIR/configs/chains/C/genesis.json" << EOF
{
  "config": {
    "chainId": $CHAIN_ID,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "muirGlacierBlock": 0,
    "subnetEVMTimestamp": 0,
    "feeConfig": {
      "gasLimit": 12000000,
      "targetBlockRate": 2,
      "minBaseFee": 25000000000,
      "targetGas": 60000000,
      "baseFeeChangeDenominator": 36,
      "minBlockGasCost": 0,
      "maxBlockGasCost": 1000000,
      "blockGasCostStep": 200000
    }
  },
  "nonce": "0x0",
  "timestamp": "0x0",
  "extraData": "0x",
  "gasLimit": "0xb71b00",
  "difficulty": "0x0",
  "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "coinbase": "0x0000000000000000000000000000000000000000",
  "alloc": {
    "$TREASURY": {
      "balance": "0x193e5939a08ce9dbd480000000"
    }
  },
  "number": "0x0",
  "gasUsed": "0x0",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000"
}
EOF

# Generate staking keys
echo "Generating staking keys..."
if [ ! -f "$DATA_DIR/staking/staker.key" ]; then
    openssl genrsa -out "$DATA_DIR/staking/staker.key" 4096 2>/dev/null
    openssl rsa -in "$DATA_DIR/staking/staker.key" -pubout -out "$DATA_DIR/staking/staker.pub" 2>/dev/null

    # Generate certificate
    openssl req -new -x509 -key "$DATA_DIR/staking/staker.key" \
        -out "$DATA_DIR/staking/staker.crt" -days 365 \
        -subj "/C=US/ST=State/L=City/O=Lux/CN=luxnode" 2>/dev/null

    cp "$DATA_DIR/staking/staker.key" "$DATA_DIR/staking/signer.key"
fi

# Link the existing database (read-only mount)
echo "Linking existing blockchain database..."
ln -sf "$EXISTING_DB/db/pebbledb" "$DATA_DIR/db/C"

# Launch the node with existing database
echo "Starting node with existing blockchain data..."
echo ""

exec "$LUXD" \
    --network-id=$CHAIN_ID \
    --db-dir="$DATA_DIR/db" \
    --chain-config-dir="$DATA_DIR/configs/chains" \
    --plugin-dir="$DATA_DIR/plugins" \
    --staking-tls-cert-file="$DATA_DIR/staking/staker.crt" \
    --staking-tls-key-file="$DATA_DIR/staking/staker.key" \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-port=9631 \
    --log-level=info \
    --api-admin-enabled=true \
    --api-eth-enabled=true \
    --api-web3-enabled=true \
    --api-debug-enabled=true \
    --api-personal-enabled=true \
    --index-enabled=true \
    --vm-manager-enabled=true