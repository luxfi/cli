#!/bin/bash
set -e

echo "=== Direct Lux Node Launch with Subnet Database ==="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
LUXD="/home/z/work/lux/node/build/luxd"
DATA_DIR="/home/z/.luxd-mainnet-96369"
SUBNET_DB="/home/z/.avalanche-cli/runs/network_original_subnet/node1/chains/2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB"
BLOCKCHAIN_ID="2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB"
CCHAIN_DIR="$DATA_DIR/chains/C"

# Check luxd binary exists
if [ ! -f "$LUXD" ]; then
    echo -e "${RED}Error: luxd binary not found at $LUXD${NC}"
    echo "Building luxd..."
    cd /home/z/work/lux/node
    ./scripts/build.sh
fi

# Kill any existing processes
echo -e "${YELLOW}Step 1: Cleaning up existing processes...${NC}"
pkill -f luxd || true
sleep 2
echo -e "${GREEN}✓ Cleaned up existing processes${NC}"

# Clean up old data
echo -e "${YELLOW}Step 2: Preparing data directory...${NC}"
if [ -d "$DATA_DIR" ]; then
    echo "Backing up existing data to $DATA_DIR.backup"
    mv "$DATA_DIR" "$DATA_DIR.backup.$(date +%s)"
fi
mkdir -p "$DATA_DIR"
echo -e "${GREEN}✓ Created data directory${NC}"

# Copy subnet database to C-Chain location
echo -e "${YELLOW}Step 3: Copying subnet database to C-Chain location...${NC}"
if [ ! -d "$SUBNET_DB" ]; then
    echo -e "${RED}Error: Subnet database not found at $SUBNET_DB${NC}"
    exit 1
fi

# Check database size
DB_SIZE=$(du -sh "$SUBNET_DB" | cut -f1)
echo "Subnet database size: $DB_SIZE"

# Create C-Chain directory structure
mkdir -p "$CCHAIN_DIR"

# Copy the entire subnet database to C-Chain location
echo "Copying database (this may take a minute)..."
cp -r "$SUBNET_DB"/* "$CCHAIN_DIR/" 2>/dev/null || true

# Also check for and copy any leveldb format if it exists
if [ -d "$SUBNET_DB/v0.8.0-rc.3" ]; then
    echo "Found versioned database, copying..."
    cp -r "$SUBNET_DB/v0.8.0-rc.3"/* "$CCHAIN_DIR/"
fi

# Verify copy
CCHAIN_SIZE=$(du -sh "$CCHAIN_DIR" | cut -f1)
echo "C-Chain database size: $CCHAIN_SIZE"
echo -e "${GREEN}✓ Copied subnet database to C-Chain location${NC}"

# Create genesis file for C-Chain
echo -e "${YELLOW}Step 4: Creating C-Chain genesis configuration...${NC}"
cat > "$DATA_DIR/genesis.json" <<EOF
{
  "config": {
    "chainId": 96369,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "muirGlacierBlock": 0,
    "berlinBlock": 0,
    "londonBlock": 0,
    "arrowGlacierBlock": 0,
    "cancunBlock": 0,
    "feeConfig": {
      "gasLimit": 20000000,
      "minBaseFee": 1000000000,
      "targetGas": 100000000,
      "baseFeeChangeDenominator": 48,
      "minBlockGasCost": 0,
      "maxBlockGasCost": 10000000,
      "targetBlockRate": 2,
      "blockGasCostStep": 500000
    }
  },
  "nonce": "0x0",
  "timestamp": "0x66cc9930",
  "gasLimit": "0x1312d00",
  "difficulty": "0x0",
  "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "coinbase": "0x0000000000000000000000000000000000000000",
  "alloc": {},
  "number": "0x0",
  "gasUsed": "0x0",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "baseFeePerGas": "0x3b9aca00"
}
EOF
echo -e "${GREEN}✓ Created genesis configuration${NC}"

# Create C-Chain config
echo -e "${YELLOW}Step 5: Creating C-Chain configuration...${NC}"
mkdir -p "$DATA_DIR/configs/chains/C"
cat > "$DATA_DIR/configs/chains/C/config.json" <<EOF
{
  "snowman-api-enabled": false,
  "coreth-admin-api-enabled": true,
  "eth-apis": ["eth", "eth-filter", "net", "web3", "internal-eth", "internal-blockchain", "internal-transaction", "internal-debug", "debug", "debug-tracer", "admin", "lux"],
  "pruning-enabled": false,
  "local-txs-enabled": true,
  "api-max-duration": 0,
  "api-max-blocks-per-request": 0,
  "allow-unfinalized-queries": true,
  "allow-unprotected-txs": true,
  "log-level": "debug",
  "state-sync-enabled": false,
  "offline-pruning-enabled": false,
  "offline-pruning-data-directory": "",
  "max-block-history-lookback": 0,
  "continuous-profiler-enabled": false
}
EOF
echo -e "${GREEN}✓ Created C-Chain configuration${NC}"

# Start luxd
echo -e "${YELLOW}Step 6: Starting luxd with C-Chain...${NC}"
nohup $LUXD \
    --network-id=96369 \
    --data-dir="$DATA_DIR" \
    --http-port=9630 \
    --log-level=info \
    --poa-mode-enabled \
    --poa-single-node-mode \
    --skip-bootstrap \
    --index-enabled \
    --public-ip=127.0.0.1 \
    --http-host=0.0.0.0 \
    --http-allowed-hosts="*" \
    --api-admin-enabled \
    > "$DATA_DIR/node.log" 2>&1 &

LUXD_PID=$!
echo "Started luxd with PID: $LUXD_PID"
echo -e "${GREEN}✓ Started luxd${NC}"

# Wait for node to start
echo -e "${YELLOW}Step 7: Waiting for node initialization...${NC}"
for i in {1..30}; do
    if curl -s -X POST http://localhost:9630/ext/health \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","method":"health.health","params":[],"id":1}' 2>/dev/null | grep -q healthy; then
        echo -e "${GREEN}✓ Node is healthy${NC}"
        break
    fi
    echo -n "."
    sleep 1
done
echo ""

# Check C-Chain status
echo -e "${YELLOW}Step 8: Checking C-Chain status...${NC}"

# Try to get block number
BLOCK_HEIGHT=$(curl -s -X POST http://localhost:9630/ext/bc/C/rpc \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    2>/dev/null | jq -r '.result' 2>/dev/null || echo "0x0")

if [ "$BLOCK_HEIGHT" != "0x0" ] && [ "$BLOCK_HEIGHT" != "null" ] && [ ! -z "$BLOCK_HEIGHT" ]; then
    DECIMAL_HEIGHT=$((16#${BLOCK_HEIGHT#0x}))
    echo -e "${GREEN}C-Chain block height: $DECIMAL_HEIGHT${NC}"

    if [ "$DECIMAL_HEIGHT" -eq 0 ]; then
        echo ""
        echo -e "${YELLOW}C-Chain is at genesis. Starting runtime replay to import subnet blocks...${NC}"

        # Trigger runtime replay
        echo "Triggering replay from subnet RPC..."
        REPLAY_RESPONSE=$(curl -s -X POST http://localhost:9630/ext/bc/C/admin \
            -H "Content-Type: application/json" \
            -d '{
                "jsonrpc": "2.0",
                "method": "lux_startReplay",
                "params": [{
                    "sourceRPC": "https://api.lux.network",
                    "startBlock": 1,
                    "endBlock": 0,
                    "continuous": true,
                    "useSnapshot": false
                }],
                "id": 1
            }' 2>/dev/null)

        echo "Replay response: $REPLAY_RESPONSE"

        # Check replay status
        sleep 5
        REPLAY_STATUS=$(curl -s -X POST http://localhost:9630/ext/bc/C/admin \
            -H "Content-Type: application/json" \
            -d '{"jsonrpc":"2.0","method":"lux_replayStatus","params":[],"id":1}' 2>/dev/null)

        echo "Replay status: $REPLAY_STATUS"
    fi
else
    echo -e "${YELLOW}C-Chain may still be initializing or needs configuration${NC}"
fi

# Display final information
echo ""
echo "=========================================="
echo -e "${GREEN}Lux node started successfully!${NC}"
echo ""
echo "Configuration:"
echo "  Data directory: $DATA_DIR"
echo "  Log file: $DATA_DIR/node.log"
echo "  Network ID: 96369"
echo "  PID: $LUXD_PID"
echo ""
echo "API Endpoints:"
echo "  Health: http://localhost:9630/ext/health"
echo "  Info: http://localhost:9630/ext/info"
echo "  C-Chain RPC: http://localhost:9630/ext/bc/C/rpc"
echo "  C-Chain WebSocket: ws://localhost:9630/ext/bc/C/ws"
echo "  Admin API: http://localhost:9630/ext/bc/C/admin"
echo ""
echo "Commands:"
echo "  View logs: tail -f $DATA_DIR/node.log"
echo "  Check block height: curl -X POST http://localhost:9630/ext/bc/C/rpc -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}'"
echo "  Start replay: curl -X POST http://localhost:9630/ext/bc/C/admin -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"method\":\"lux_startReplay\",\"params\":[{\"sourceRPC\":\"https://api.lux.network\",\"startBlock\":1,\"endBlock\":0,\"continuous\":true}],\"id\":1}'"
echo "  Check replay: curl -X POST http://localhost:9630/ext/bc/C/admin -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"method\":\"lux_replayStatus\",\"params\":[],\"id\":1}'"
echo "  Stop node: kill $LUXD_PID"
echo "=========================================="
echo ""
echo -e "${GREEN}=== Deployment Complete ===${NC}"