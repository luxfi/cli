#!/bin/bash

# Import historical chain data into C-Chain

# Source data (LUX mainnet subnet EVM with ~1M blocks)
SOURCE_DB="/home/z/.luxd/db.backup.1752452721/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ/db/pebbledb"

# Get latest network run
NETWORK_DIR=$(ls -td ~/.avalanche-cli/runs/network_* | head -1)

# Target C-Chain database directory for both nodes
NODE1_CCHAIN_DB="${NETWORK_DIR}/NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg/db/2CA6j5zYzasynPsFeNoqWkmTCt3VScMvXUZHbfDJ8k3oGzAPtU/db/pebbledb"
NODE2_CCHAIN_DB="${NETWORK_DIR}/NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ/db/2CA6j5zYzasynPsFeNoqWkmTCt3VScMvXUZHbfDJ8k3oGzAPtU/db/pebbledb"

echo "Import Chain Data Script"
echo "========================"
echo "Source DB: $SOURCE_DB"
echo "Target Network: $NETWORK_DIR"
echo ""

# Verify source exists
if [ ! -d "$SOURCE_DB" ]; then
    echo "Error: Source database not found at $SOURCE_DB"
    exit 1
fi

# Create target directories if they don't exist
echo "Creating target directories..."
mkdir -p "$(dirname "$NODE1_CCHAIN_DB")"
mkdir -p "$(dirname "$NODE2_CCHAIN_DB")"

# Remove existing C-Chain databases if they exist
if [ -d "$NODE1_CCHAIN_DB" ]; then
    echo "Removing existing C-Chain database for Node 1..."
    rm -rf "$NODE1_CCHAIN_DB"
fi

if [ -d "$NODE2_CCHAIN_DB" ]; then
    echo "Removing existing C-Chain database for Node 2..."
    rm -rf "$NODE2_CCHAIN_DB"
fi

# Copy the historical data to both nodes
echo "Copying historical data to Node 1..."
cp -r "$SOURCE_DB" "$NODE1_CCHAIN_DB"

echo "Copying historical data to Node 2..."
cp -r "$SOURCE_DB" "$NODE2_CCHAIN_DB"

# Set proper permissions
echo "Setting permissions..."
chmod -R 755 "${NETWORK_DIR}/NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg/db/"
chmod -R 755 "${NETWORK_DIR}/NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ/db/"

echo ""
echo "Import complete! Historical chain data has been imported to C-Chain."
echo "You can now start the network with: avalanche network start"