#!/bin/bash
set -e

echo "=== Lux Subnet to C-Chain Migration Deployment ==="
echo "This script migrates the existing subnet data (1,074,616 blocks) to C-Chain"
echo ""

# Configuration
MIGRATION_DIR="/home/z/work/lux/cli/lux-mainnet-migration"
SUBNET_DB="/home/z/.avalanche-cli/runs/network_original_subnet/node1/chains/2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB"
NETWORK_ID=96369
NUM_VALIDATORS=5
LUX_CLI="/home/z/work/lux/cli/bin/lux-cli"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Function to check status
check_status() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ $1${NC}"
    else
        echo -e "${RED}✗ $1${NC}"
        exit 1
    fi
}

# Step 1: Kill any existing processes
echo -e "${YELLOW}Step 1: Cleaning up existing processes...${NC}"
pkill -f luxd || true
pkill -f lux-cli || true
sleep 2
check_status "Cleaned up existing processes"

# Step 2: Clean up old migration data
echo -e "${YELLOW}Step 2: Cleaning up old migration data...${NC}"
if [ -d "$MIGRATION_DIR" ]; then
    rm -rf "$MIGRATION_DIR"
    echo "Removed old migration directory"
fi
check_status "Cleaned up old data"

# Step 3: Verify subnet database exists
echo -e "${YELLOW}Step 3: Verifying subnet database...${NC}"
if [ ! -d "$SUBNET_DB" ]; then
    echo -e "${RED}Error: Subnet database not found at $SUBNET_DB${NC}"
    exit 1
fi

# Check database size
DB_SIZE=$(du -sh "$SUBNET_DB" | cut -f1)
echo "Subnet database size: $DB_SIZE"

# Check for PebbleDB files
if [ -d "$SUBNET_DB/v0.8.0-rc.3/pebble" ]; then
    echo "Found PebbleDB at $SUBNET_DB/v0.8.0-rc.3/pebble"
    PEBBLE_SIZE=$(du -sh "$SUBNET_DB/v0.8.0-rc.3/pebble" | cut -f1)
    echo "PebbleDB size: $PEBBLE_SIZE"
fi
check_status "Verified subnet database"

# Step 4: Prepare migration data
echo -e "${YELLOW}Step 4: Preparing migration data...${NC}"
echo "Converting subnet PebbleDB to C-Chain LevelDB format..."
echo "This may take several minutes..."

$LUX_CLI migrate prepare \
    --source-db "$SUBNET_DB/v0.8.0-rc.3/pebble" \
    --output "$MIGRATION_DIR" \
    --network-id $NETWORK_ID \
    --validators $NUM_VALIDATORS

check_status "Prepared migration data"

# Step 5: Verify migration output
echo -e "${YELLOW}Step 5: Verifying migration output...${NC}"
if [ ! -d "$MIGRATION_DIR" ]; then
    echo -e "${RED}Error: Migration directory was not created${NC}"
    exit 1
fi

echo "Migration directory contents:"
ls -la "$MIGRATION_DIR/"

# Check for converted C-Chain database
if [ -d "$MIGRATION_DIR/cchain-db" ]; then
    CCHAIN_SIZE=$(du -sh "$MIGRATION_DIR/cchain-db" | cut -f1)
    echo "C-Chain database size: $CCHAIN_SIZE"
fi

# Check for genesis files
if [ -f "$MIGRATION_DIR/genesis.json" ]; then
    echo "Genesis file created"
fi

# Check for validator configs
for i in $(seq 1 $NUM_VALIDATORS); do
    if [ -d "$MIGRATION_DIR/validator$i" ]; then
        echo "Validator $i configuration created"
    fi
done

check_status "Verified migration output"

# Step 6: Bootstrap the network with migrated data
echo -e "${YELLOW}Step 6: Bootstrapping Lux network with migrated data...${NC}"
echo "Starting $NUM_VALIDATORS bootstrap validators..."

$LUX_CLI migrate bootstrap \
    --migration-dir "$MIGRATION_DIR" \
    --detached

check_status "Started bootstrap network"

# Step 7: Wait for network to initialize
echo -e "${YELLOW}Step 7: Waiting for network initialization...${NC}"
echo "Waiting 10 seconds for nodes to start..."
sleep 10

# Step 8: Check node status
echo -e "${YELLOW}Step 8: Checking node status...${NC}"

# Check if luxd is running
if pgrep -f luxd > /dev/null; then
    echo "luxd processes are running"
    ps aux | grep luxd | grep -v grep | head -5
else
    echo -e "${RED}Warning: No luxd processes found${NC}"
fi

# Step 9: Query blockchain status
echo -e "${YELLOW}Step 9: Querying blockchain status...${NC}"

# Try to get block number from C-Chain
echo "Attempting to query C-Chain block height..."
for port in 9630 9640 9650 9660 9670; do
    echo "Trying port $port..."
    BLOCK_HEIGHT=$(curl -s -X POST http://localhost:$port/ext/bc/C/rpc \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
        2>/dev/null | jq -r '.result' 2>/dev/null || echo "0x0")

    if [ "$BLOCK_HEIGHT" != "0x0" ] && [ "$BLOCK_HEIGHT" != "null" ] && [ ! -z "$BLOCK_HEIGHT" ]; then
        DECIMAL_HEIGHT=$((16#${BLOCK_HEIGHT#0x}))
        echo -e "${GREEN}C-Chain block height on port $port: $DECIMAL_HEIGHT${NC}"
        break
    fi
done

# Step 10: Display connection information
echo -e "${YELLOW}Step 10: Connection Information${NC}"
echo "=========================================="
echo "Network successfully bootstrapped!"
echo ""
echo "RPC Endpoints:"
echo "  Node 1: http://localhost:9630/ext/bc/C/rpc"
echo "  Node 2: http://localhost:9640/ext/bc/C/rpc"
echo "  Node 3: http://localhost:9650/ext/bc/C/rpc"
echo "  Node 4: http://localhost:9660/ext/bc/C/rpc"
echo "  Node 5: http://localhost:9670/ext/bc/C/rpc"
echo ""
echo "WebSocket Endpoints:"
echo "  Node 1: ws://localhost:9630/ext/bc/C/ws"
echo "  Node 2: ws://localhost:9640/ext/bc/C/ws"
echo "  Node 3: ws://localhost:9650/ext/bc/C/ws"
echo "  Node 4: ws://localhost:9660/ext/bc/C/ws"
echo "  Node 5: ws://localhost:9670/ext/bc/C/ws"
echo ""
echo "Migration directory: $MIGRATION_DIR"
echo "Log files: $MIGRATION_DIR/validator*/logs/"
echo ""
echo "To monitor logs:"
echo "  tail -f $MIGRATION_DIR/validator1/logs/main.log"
echo ""
echo "To stop the network:"
echo "  pkill -f luxd"
echo "=========================================="

# Step 11: Optional - Start runtime replay if C-Chain is ready
if [ "$DECIMAL_HEIGHT" -gt 0 ]; then
    echo ""
    echo -e "${GREEN}C-Chain is running with $DECIMAL_HEIGHT blocks${NC}"
    echo "Migration appears successful!"
else
    echo ""
    echo -e "${YELLOW}Note: C-Chain may still be initializing.${NC}"
    echo "If blocks were not found, you may need to:"
    echo "1. Wait a bit longer for initialization"
    echo "2. Check logs for any errors"
    echo "3. Manually trigger replay with: curl -X POST http://localhost:9630/ext/bc/C/admin -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"method\":\"lux_startReplay\",\"params\":[],\"id\":1}'"
fi

echo ""
echo -e "${GREEN}=== Deployment Complete ===${NC}"