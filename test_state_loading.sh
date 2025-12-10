#!/bin/bash

# Test script for loading existing subnet state with lux-cli
# This demonstrates how to load the 9.3GB database with 1,074,616 blocks

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
LUX_CLI="${SCRIPT_DIR}/bin/lux"

# Check if lux-cli binary exists
if [ ! -x "$LUX_CLI" ]; then
    echo "Error: lux CLI not found at $LUX_CLI"
    echo "Please run: make build"
    exit 1
fi

# Path to existing database with 1M+ blocks
DEFAULT_DB="/home/z/.lux-cli/runs/mainnet-regenesis/node1/chains/2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB/db"

# Check if database exists
if [ -d "$DEFAULT_DB" ]; then
    DB_SIZE=$(du -sh "$DEFAULT_DB" | cut -f1)
    echo "Found existing database at: $DEFAULT_DB"
    echo "Database size: $DB_SIZE"
    echo ""
fi

echo "Starting network with existing subnet state..."
echo ""

# Option 1: Start with automatic detection of default database
echo "Option 1: Start with automatic detection (default database will be loaded if found)"
echo "Command: $LUX_CLI network start"
echo ""

# Option 2: Explicitly specify the subnet database path
echo "Option 2: Explicitly specify the database path"
echo "Command: $LUX_CLI network start --subnet-state-path=\"$DEFAULT_DB\" --blockchain-id=\"2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB\""
echo ""

# Option 3: Use state-path for more complex scenarios
echo "Option 3: Use state-path for chaindata directory"
echo "Command: $LUX_CLI network start --state-path=\"/home/z/work/lux/state/chaindata/lux-mainnet-96369\""
echo ""

# Actually run with automatic detection
read -p "Press Enter to start network with automatic state loading..."
$LUX_CLI network start

echo ""
echo "Network started. The existing state should now be available."
echo "Check logs to verify that the database was loaded successfully."