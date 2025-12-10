#!/bin/bash
set -e

echo "==================================================================="
echo "Starting Single Node LUX with Symlinked Migrated Data"
echo "==================================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
NODE_DIR="/home/z/work/lux/node"
CLI_DIR="/home/z/work/lux/cli"
# Use the actual 7.2GB database
SOURCE_DB="/home/z/work/lux/state-broken/chaindata/lux-mainnet-96369/db_final_backup_20251002_191045/pebbledb"
# The VM expects the database at this specific path
EXPECTED_PATH="/home/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb"
DATA_DIR="/tmp/luxd_symlink"
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

echo -e "${YELLOW}Step 2: Setting up symlink to migrated database...${NC}"
# Remove old symlink or directory if it exists
if [ -L "${EXPECTED_PATH}" ]; then
    rm "${EXPECTED_PATH}"
    echo "Removed existing symlink"
elif [ -d "${EXPECTED_PATH}" ]; then
    echo "Found existing directory at ${EXPECTED_PATH}, backing it up..."
    mv "${EXPECTED_PATH}" "${EXPECTED_PATH}.backup.$(date +%s)"
    echo "Backed up existing directory"
fi

# Create directory structure if needed
mkdir -p $(dirname "${EXPECTED_PATH}")

# Create symlink to the actual database
ln -s "${SOURCE_DB}" "${EXPECTED_PATH}"
echo "Created symlink: ${EXPECTED_PATH} -> ${SOURCE_DB}"

# Verify symlink
if [ -L "${EXPECTED_PATH}" ]; then
    echo -e "${GREEN}Symlink created successfully${NC}"
    ls -la "${EXPECTED_PATH}"
else
    echo -e "${RED}Failed to create symlink${NC}"
    exit 1
fi

# Check database size through symlink
DB_SIZE=$(du -sh "${EXPECTED_PATH}" 2>/dev/null | cut -f1)
echo "Database size (via symlink): ${DB_SIZE}"

# Check for BadgerDB files
if ls "${EXPECTED_PATH}"/*.vlog 1> /dev/null 2>&1; then
    echo -e "${GREEN}Confirmed: BadgerDB format accessible via symlink${NC}"
else
    echo -e "${YELLOW}Warning: No .vlog files found${NC}"
fi

echo -e "${YELLOW}Step 3: Creating configuration file...${NC}"
# Create a config file with BadgerDB settings
cat > ${DATA_DIR}/config.json << EOF
{
  "network-id": "96369",
  "http-port": ${HTTP_PORT},
  "staking-port": ${STAKING_PORT},
  "http-host": "127.0.0.1",
  "staking-host": "",
  "log-level": "info",
  "log-format": "plain",
  "db-type": "badgerdb",
  "c-chain-db-type": "badgerdb",
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

echo -e "${YELLOW}Step 4: Starting luxd with dev mode...${NC}"
cd ${NODE_DIR}

# Start luxd with the dev flag - it should detect the database at the expected path
nohup ./build/luxd \
    --data-dir=${DATA_DIR} \
    --config-file=${DATA_DIR}/config.json \
    --dev \
    --enable-automining \
    --force-ignore-checksum \
    > ${DATA_DIR}/logs/luxd.log 2>&1 &

LUXD_PID=$!
echo "luxd started with PID: ${LUXD_PID}"

echo -e "${YELLOW}Step 5: Waiting for node to start...${NC}"
# Wait longer for the node to process the migrated data
sleep 20

# Check if process is still running
if ! ps -p ${LUXD_PID} > /dev/null; then
    echo -e "${RED}luxd failed to start. Showing recent logs:${NC}"
    tail -n 100 ${DATA_DIR}/logs/luxd.log
    exit 1
fi

echo -e "${YELLOW}Step 6: Checking node health...${NC}"
for i in {1..60}; do
    if curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"info.getNodeID","params":{}}' \
        -H 'content-type:application/json' http://127.0.0.1:${HTTP_PORT}/ext/info > /dev/null 2>&1; then
        echo -e "${GREEN}Node is healthy!${NC}"
        break
    fi
    echo -n "."
    sleep 2
done

echo -e "\n${YELLOW}Step 7: Checking C-Chain status...${NC}"
# Get C-Chain block number
BLOCK_HEIGHT=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    -H 'Content-Type: application/json' \
    http://127.0.0.1:${HTTP_PORT}/ext/bc/C/rpc | jq -r '.result' | xargs printf "%d\n" 2>/dev/null || echo "0")

echo -e "${GREEN}C-Chain Block Height: ${BLOCK_HEIGHT}${NC}"

if [ ${BLOCK_HEIGHT} -ge 1074616 ]; then
    echo -e "${GREEN}✓ SUCCESS! Block height matches expected value (1,074,616)!${NC}"
    echo "The migrated blockchain data is being read correctly!"
else
    echo -e "${YELLOW}Block height is ${BLOCK_HEIGHT}, expected at least 1,074,616${NC}"
    echo "Checking if data is still loading..."

    # Check again after a delay
    sleep 10
    BLOCK_HEIGHT=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
        -H 'Content-Type: application/json' \
        http://127.0.0.1:${HTTP_PORT}/ext/bc/C/rpc | jq -r '.result' | xargs printf "%d\n" 2>/dev/null || echo "0")
    echo "Updated block height: ${BLOCK_HEIGHT}"
fi

echo -e "${YELLOW}Step 8: Checking treasury balance...${NC}"
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

    # Check if it matches expected value (113.2M LUX)
    EXPECTED_BALANCE="113200000000000000000000000"  # 113.2M LUX in Wei
    if [ "${BALANCE_WEI}" = "${EXPECTED_BALANCE}" ]; then
        echo -e "${GREEN}✓ SUCCESS! Treasury balance matches expected value (113.2M LUX)!${NC}"
    fi
else
    echo -e "${YELLOW}Could not retrieve balance${NC}"
fi

echo -e "\n${GREEN}==================================================================="
echo "Single Node Setup Complete!"
echo "==================================================================="
echo "Data Directory: ${DATA_DIR}"
echo "Logs: ${DATA_DIR}/logs/luxd.log"
echo "Process PID: ${LUXD_PID}"
echo "Database Symlink: ${EXPECTED_PATH} -> ${SOURCE_DB}"
echo ""
echo "RPC Endpoints:"
echo "  Info API: http://localhost:${HTTP_PORT}/ext/info"
echo "  C-Chain RPC: http://localhost:${HTTP_PORT}/ext/bc/C/rpc"
echo "  C-Chain WebSocket: ws://localhost:${HTTP_PORT}/ext/bc/C/ws"
echo ""
echo "Test Commands:"
echo "  # Get block height:"
echo "  curl -s -X POST --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}' \\"
echo "    -H 'Content-Type: application/json' http://localhost:${HTTP_PORT}/ext/bc/C/rpc | jq"
echo ""
echo "  # Get balance:"
echo "  curl -s -X POST --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"0x9011E888251AB053B7bD1cdB598Db4f9DEd94714\",\"latest\"],\"id\":1}' \\"
echo "    -H 'Content-Type: application/json' http://localhost:${HTTP_PORT}/ext/bc/C/rpc | jq"
echo ""
echo "To monitor logs:"
echo "  tail -f ${DATA_DIR}/logs/luxd.log"
echo ""
echo "To stop the node:"
echo "  kill ${LUXD_PID}"
echo "  rm ${EXPECTED_PATH}  # Remove symlink"
echo "==================================================================="
echo -e "${NC}"