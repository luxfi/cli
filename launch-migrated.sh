#!/bin/bash
# LUX Migrated Chain Launcher

LUXD="/home/z/work/lux/node/build/luxd"

if [ ! -f "$LUXD" ]; then
    echo "Building luxd..."
    cd /home/z/work/lux/node && ./scripts/build.sh
fi

echo "Starting migrated LUX node..."
echo "Chain ID: 96369"
echo "Database: /tmp/lux-c-chain-import"
echo "Treasury: 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714"
echo ""

# Launch with migrated database
exec "$LUXD" \
    --config-file=/home/z/.luxd-migrated/config.json \
    --db-dir=/tmp/lux-c-chain-import \
    --chain-data-dir=/tmp/lux-c-chain-import/chaindata \
    --network-id=96369 \
    --http-port=9630 \
    --staking-port=9631 \
    --staking-enabled=false \
    --health-check-frequency=5s \
    --api-admin-enabled \
    --api-eth-enabled \
    --api-web3-enabled \
    --api-debug-enabled \
    --api-personal-enabled \
    --api-txpool-enabled \
    --api-net-enabled \
    --log-level=info
