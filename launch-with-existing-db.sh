#!/bin/bash
# Launch LUX node with existing blockchain database (read-only)

LUXD="/home/z/work/lux/node/build/luxd"
EXISTING_DB="/home/z/work/lux/state/chaindata/lux-mainnet-96369"
GENESIS="/home/z/.luxd-migrated/configs/chains/C/genesis.json"

echo "=== LUX Mainnet with Existing Blockchain Data ==="
echo "Chain ID: 96369"
echo "Network ID: 96369"
echo "Treasury: 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714 (2T LUX)"
echo "Using existing database: $EXISTING_DB"
echo "Expected blocks: 850,870"
echo ""

# Check if database exists
if [ ! -d "$EXISTING_DB/db/pebbledb" ]; then
    echo "ERROR: Database not found at $EXISTING_DB/db/pebbledb"
    exit 1
fi

# Check database size
DB_SIZE=$(du -sh "$EXISTING_DB/db/pebbledb" | cut -f1)
echo "Database size: $DB_SIZE (expected ~7.2GB)"
echo ""

# Create a symlink to use the existing database (read-only access)
DATA_DIR="/tmp/lux-existing-db-$$"
mkdir -p "$DATA_DIR"

# Link the existing database instead of copying
ln -sf "$EXISTING_DB" "$DATA_DIR/chaindata"

echo "Starting node with existing blockchain data..."
echo "Data directory: $DATA_DIR"
echo ""

# Launch luxd with the existing blockchain database
exec "$LUXD" \
    --dev \
    --network-id=96369 \
    --db-dir="$DATA_DIR" \
    --chain-data-dir="$DATA_DIR/chaindata" \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-port=9631 \
    --api-admin-enabled=true \
    --index-enabled=true \
    --log-level=info