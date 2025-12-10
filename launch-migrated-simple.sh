#!/bin/bash
# Simple LUX Migrated Chain Launcher

LUXD="/home/z/work/lux/node/build/luxd"
DB_DIR="/tmp/lux-c-chain-import"
GENESIS="/home/z/.luxd-migrated/configs/chains/C/genesis.json"

echo "Starting migrated LUX node..."
echo "Chain ID: 96369"
echo "Database: $DB_DIR"
echo "Genesis: $GENESIS"
echo "Treasury: 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714"
echo ""

# Create data directory structure
mkdir -p "$DB_DIR/network-96369"

# Launch with minimal configuration
exec "$LUXD" \
    --network-id=96369 \
    --db-dir="$DB_DIR" \
    --http-port=9630 \
    --http-host=0.0.0.0 \
    --log-level=info