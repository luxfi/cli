#!/bin/bash
set -e

echo "==================================================================="
echo "Starting Proper Single Node LUX with Migrated Data"
echo "==================================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
NODE_DIR="/home/z/work/lux/node"
CLI_DIR="/home/z/work/lux/cli"
# Use the actual 7.2GB database from state-broken
SOURCE_DB="/home/z/work/lux/state-broken/chaindata/lux-mainnet-96369/db_final_backup_20251002_191045/pebbledb"
DATA_DIR="/tmp/luxd_single_proper"
HTTP_PORT=9630
STAKING_PORT=9631

echo -e "${YELLOW}Step 1: Cleaning up old processes and data...${NC}"
# Kill any existing luxd processes
pkill -9 luxd 2>/dev/null || true
sleep 2

# Clean up old data
rm -rf ${DATA_DIR}
mkdir -p ${DATA_DIR}
mkdir -p ${DATA_DIR}/db
mkdir -p ${DATA_DIR}/chainData
mkdir -p ${DATA_DIR}/logs
mkdir -p ${DATA_DIR}/staking

echo -e "${YELLOW}Step 2: Checking source database...${NC}"
if [ ! -d "${SOURCE_DB}" ]; then
    echo -e "${RED}Source database not found at ${SOURCE_DB}${NC}"
    exit 1
fi

# Check database size
DB_SIZE=$(du -sh ${SOURCE_DB} 2>/dev/null | cut -f1)
echo "Source database size: ${DB_SIZE}"

# Count number of files to ensure complete copy
FILE_COUNT=$(find ${SOURCE_DB} -type f | wc -l)
echo "Number of files in source database: ${FILE_COUNT}"

echo -e "${YELLOW}Step 3: Copying migrated PebbleDB database (this may take a few minutes)...${NC}"
# Create the C-Chain data directory structure
CCHAIN_DB="${DATA_DIR}/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ/db"
mkdir -p ${CCHAIN_DB}

# Copy the entire PebbleDB database
echo "Copying PebbleDB database..."
cp -r ${SOURCE_DB} ${CCHAIN_DB}/
mv ${CCHAIN_DB}/pebbledb ${CCHAIN_DB}/chaindb

# Verify the copy
COPIED_FILES=$(find ${CCHAIN_DB}/chaindb -type f | wc -l)
echo "Files copied: ${COPIED_FILES} of ${FILE_COUNT}"

if [ ${COPIED_FILES} -ne ${FILE_COUNT} ]; then
    echo -e "${RED}Warning: Not all files were copied!${NC}"
fi

COPIED_SIZE=$(du -sh ${CCHAIN_DB}/chaindb 2>/dev/null | cut -f1)
echo "Copied database size: ${COPIED_SIZE}"

echo -e "${YELLOW}Step 4: Preparing staking keys (using ephemeral mode)...${NC}"
# We'll use ephemeral staking in the config to avoid certificate issues
mkdir -p ${DATA_DIR}/staking

echo -e "${YELLOW}Step 5: Creating configuration file...${NC}"
# Create a config file for proper settings
cat > ${DATA_DIR}/config.json << EOF
{
  "network-id": "96369",
  "http-port": ${HTTP_PORT},
  "staking-port": ${STAKING_PORT},
  "http-host": "127.0.0.1",
  "staking-host": "",
  "log-level": "info",
  "log-format": "plain",
  "db-type": "pebbledb",
  "c-chain-db-type": "pebbledb",
  "index-enabled": true,
  "api-admin-enabled": true,
  "api-health-enabled": true,
  "api-info-enabled": true,
  "api-metrics-enabled": true,
  "consensus-sample-size": 1,
  "consensus-quorum-size": 1,
  "consensus-preference-quorum-size": 1,
  "consensus-confidence-quorum-size": 1,
  "sybil-protection-enabled": false,
  "sybil-protection-disabled-weight": 100,
  "staking-ephemeral-cert-enabled": true,
  "staking-ephemeral-signer-enabled": true,
  "network-compression-type": "none",
  "consensus-commit-threshold": 1,
  "consensus-concurrent-repolls": 1
}
EOF

echo -e "${YELLOW}Step 6: Starting luxd with dev mode for single node...${NC}"
cd ${NODE_DIR}

# Start luxd with the dev flag and additional single-node configuration
nohup ./build/luxd \
    --data-dir=${DATA_DIR} \
    --config-file=${DATA_DIR}/config.json \
    --dev \
    --enable-automining \
    --force-ignore-checksum \
    > ${DATA_DIR}/logs/luxd.log 2>&1 &

LUXD_PID=$!
echo "luxd started with PID: ${LUXD_PID}"

echo -e "${YELLOW}Step 7: Waiting for node to start...${NC}"
# Wait for the node to start
sleep 10

# Check if process is still running
if ! ps -p ${LUXD_PID} > /dev/null; then
    echo -e "${RED}luxd failed to start. Check logs:${NC}"
    tail -n 50 ${DATA_DIR}/logs/luxd.log
    exit 1
fi

echo -e "${YELLOW}Step 8: Checking node health...${NC}"
for i in {1..30}; do
    if curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"info.getNodeID","params":{}}' \
        -H 'content-type:application/json' http://127.0.0.1:${HTTP_PORT}/ext/info > /dev/null 2>&1; then
        echo -e "${GREEN}Node is healthy!${NC}"
        break
    fi
    echo -n "."
    sleep 2
done

echo -e "\n${YELLOW}Step 9: Checking C-Chain status...${NC}"
# Get C-Chain block number
BLOCK_HEIGHT=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    -H 'Content-Type: application/json' \
    http://127.0.0.1:${HTTP_PORT}/ext/bc/C/rpc | jq -r '.result' | xargs printf "%d\n" 2>/dev/null || echo "0")

echo -e "${GREEN}C-Chain Block Height: ${BLOCK_HEIGHT}${NC}"

if [ ${BLOCK_HEIGHT} -eq 1074616 ]; then
    echo -e "${GREEN}✓ Block height matches expected value!${NC}"
else
    echo -e "${YELLOW}Block height is ${BLOCK_HEIGHT}, expected 1074616${NC}"
fi

echo -e "${YELLOW}Step 10: Checking treasury balance...${NC}"
# Check treasury balance at the specific address
TREASURY_ADDRESS="0x9011E888251AB053B7bD1cdB598Db4f9DEd94714"
BALANCE_HEX=$(curl -s -X POST --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"${TREASURY_ADDRESS}\",\"latest\"],\"id\":1}" \
    -H 'Content-Type: application/json' \
    http://127.0.0.1:${HTTP_PORT}/ext/bc/C/rpc | jq -r '.result')

if [ "${BALANCE_HEX}" != "null" ] && [ "${BALANCE_HEX}" != "" ]; then
    # Convert hex to decimal and then to LUX (divide by 10^18)
    BALANCE_WEI=$(printf "%d\n" ${BALANCE_HEX} 2>/dev/null || echo "0")
    BALANCE_LUX=$(echo "scale=4; ${BALANCE_WEI} / 1000000000000000000" | bc 2>/dev/null || echo "0")
    echo -e "${GREEN}Treasury Balance: ${BALANCE_LUX} LUX${NC}"
    echo "Balance in Wei: ${BALANCE_WEI}"
else
    echo -e "${YELLOW}Could not retrieve balance${NC}"
fi

echo -e "\n${GREEN}==================================================================="
echo "Single Node Setup Complete!"
echo "==================================================================="
echo "Data Directory: ${DATA_DIR}"
echo "Logs: ${DATA_DIR}/logs/luxd.log"
echo ""
echo "RPC Endpoints:"
echo "  Info API: http://localhost:${HTTP_PORT}/ext/info"
echo "  C-Chain RPC: http://localhost:${HTTP_PORT}/ext/bc/C/rpc"
echo "  C-Chain WebSocket: ws://localhost:${HTTP_PORT}/ext/bc/C/ws"
echo ""
echo "To monitor logs:"
echo "  tail -f ${DATA_DIR}/logs/luxd.log"
echo ""
echo "To stop the node:"
echo "  kill ${LUXD_PID}"
echo "==================================================================="
echo -e "${NC}"

# Keep the script running and show logs
echo -e "\n${YELLOW}Showing live logs (Ctrl+C to exit):${NC}"
tail -f ${DATA_DIR}/logs/luxd.log