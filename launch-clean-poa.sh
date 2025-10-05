#!/bin/bash

# Clean Single Node POA Launch Script for LUX Network 96369
# This script launches a fresh standalone node without any migration data
# Perfect for immediate local development

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo ""
echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}    LUX Clean POA Node - Network ID 96369${NC}"
echo -e "${CYAN}    Fresh start - No migration data${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo ""

# Configuration
NETWORK_ID=96369
HTTP_PORT=9630
STAKING_PORT=9631

# Paths
LUXD_BIN="/home/z/work/lux/node/build/luxd"
DATA_DIR="${HOME}/.luxd-clean"
PLUGIN_DIR="${DATA_DIR}/plugins"
STAKING_DIR="${DATA_DIR}/staking"
DB_DIR="${DATA_DIR}/db/mainnet"
CHAIN_CONFIG_DIR="${DATA_DIR}/configs/chains"
LOG_FILE="${DATA_DIR}/logs/luxd-clean.log"
PID_FILE="${DATA_DIR}/luxd-clean.pid"

# Kill any existing luxd processes
echo -e "${YELLOW}Stopping any existing luxd processes...${NC}"
pkill -f luxd 2>/dev/null || true
sleep 2

# Clean up old data
if [ -d "${DATA_DIR}" ]; then
    echo -e "${YELLOW}Removing old clean data directory...${NC}"
    rm -rf "${DATA_DIR}"
fi

# Check prerequisites
echo -e "${BLUE}Checking prerequisites...${NC}"

if [ ! -f "${LUXD_BIN}" ]; then
    echo -e "${RED}✗ luxd binary not found at ${LUXD_BIN}${NC}"
    echo -e "${YELLOW}Building luxd...${NC}"
    cd /home/z/work/lux/node && ./scripts/build.sh
    echo -e "${GREEN}✓ luxd built successfully${NC}"
else
    echo -e "${GREEN}✓ luxd binary found${NC}"
fi

# Ensure directories exist
echo -e "${BLUE}Creating required directories...${NC}"
mkdir -p "${DATA_DIR}"
mkdir -p "${PLUGIN_DIR}"
mkdir -p "${STAKING_DIR}"
mkdir -p "${DB_DIR}"
mkdir -p "${CHAIN_CONFIG_DIR}/C"
mkdir -p "$(dirname ${LOG_FILE})"
echo -e "${GREEN}✓ Directories created${NC}"

# Copy EVM plugin
if [ -f "/home/z/work/lux/geth/build/geth" ]; then
    echo -e "${BLUE}Installing EVM plugin...${NC}"
    cp /home/z/work/lux/geth/build/geth "${PLUGIN_DIR}/evm"
    chmod +x "${PLUGIN_DIR}/evm"
    echo -e "${GREEN}✓ EVM plugin installed${NC}"
else
    echo -e "${RED}✗ EVM plugin not found! Building...${NC}"
    cd /home/z/work/lux/geth && ./scripts/build.sh
    cp /home/z/work/lux/geth/build/geth "${PLUGIN_DIR}/evm"
    chmod +x "${PLUGIN_DIR}/evm"
    echo -e "${GREEN}✓ EVM plugin built and installed${NC}"
fi

# Create C-Chain config (DO NOT create genesis - let platform generate it)
echo -e "${BLUE}Creating C-Chain config...${NC}"
cat > "${CHAIN_CONFIG_DIR}/C/config.json" <<'CCONFIG'
{
  "state-sync-enabled": false,
  "state-sync-skip-resume": false,
  "offline-pruning-enabled": false,
  "api-max-duration": 120000000000,
  "api-max-blocks-per-request": 0,
  "allow-unfinalized-queries": true,
  "allow-unprotected-txs": true,
  "eth-apis": [
    "eth",
    "eth-filter",
    "net",
    "web3",
    "internal-eth",
    "internal-blockchain",
    "internal-transaction",
    "internal-debug",
    "internal-account",
    "internal-personal",
    "debug-handler"
  ],
  "log-level": "info"
}
CCONFIG

echo -e "${GREEN}✓ C-Chain configuration created (genesis will be auto-generated)${NC}"

echo ""
echo -e "${CYAN}Starting luxd in clean POA mode...${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo ""

# Display configuration
echo -e "${BLUE}Configuration:${NC}"
echo -e "  Network ID:       ${GREEN}${NETWORK_ID}${NC}"
echo -e "  HTTP Port:        ${GREEN}${HTTP_PORT}${NC}"
echo -e "  Staking Port:     ${GREEN}${STAKING_PORT}${NC}"
echo -e "  Data Dir:         ${GREEN}${DATA_DIR}${NC}"
echo -e "  DB Dir:           ${GREEN}${DB_DIR}${NC}"
echo -e "  Plugin Dir:       ${GREEN}${PLUGIN_DIR}${NC}"
echo -e "  Log File:         ${GREEN}${LOG_FILE}${NC}"
echo -e "  Mode:             ${YELLOW}CLEAN START + POA${NC}"
echo ""

cd /home/z/work/lux/node

echo -e "${YELLOW}Launching luxd in clean POA mode...${NC}"
export DISABLE_MIGRATION_DETECTION=1
nohup ./build/luxd \
  --network-id=${NETWORK_ID} \
  --data-dir="${DATA_DIR}" \
  --db-dir="${DB_DIR}" \
  --log-dir="$(dirname ${LOG_FILE})" \
  --plugin-dir="${PLUGIN_DIR}" \
  --chain-config-dir="${CHAIN_CONFIG_DIR}" \
  --http-host=0.0.0.0 \
  --http-port=${HTTP_PORT} \
  --staking-port=${STAKING_PORT} \
  --public-ip=127.0.0.1 \
  --poa-single-node-mode \
  --skip-bootstrap \
  --sybil-protection-enabled=false \
  --sybil-protection-disabled-weight=100 \
  --consensus-sample-size=1 \
  --consensus-quorum-size=1 \
  --consensus-commit-threshold=1 \
  --network-peer-list-pull-gossip-frequency=2s \
  --network-health-min-conn-peers=0 \
  --network-health-max-send-fail-rate=1.0 \
  --health-check-frequency=2s \
  --http-allowed-hosts="*" \
  --http-allowed-origins="*" \
  --api-admin-enabled \
  --api-health-enabled \
  --api-info-enabled \
  --api-metrics-enabled \
  --index-enabled \
  --log-level=info \
  --log-display-level=info \
  --consensus-shutdown-timeout=60s \
  > "${LOG_FILE}" 2>&1 &

PID=$!
echo ${PID} > "${PID_FILE}"

echo -e "${GREEN}✓ luxd started with PID: ${PID}${NC}"
echo ""
echo -e "${YELLOW}Waiting for node initialization (20 seconds)...${NC}"

# Wait and show progress
for i in {1..20}; do
    sleep 1
    echo -n "."
done
echo ""

# Check if node is running
if ps -p ${PID} > /dev/null; then
    echo -e "${GREEN}✓ Node is running!${NC}"
    echo ""

    # Wait a bit more for C-Chain to initialize
    echo -e "${YELLOW}Waiting for C-Chain initialization...${NC}"
    sleep 10

    # Test C-Chain RPC
    echo -e "${BLUE}Testing C-Chain RPC endpoint...${NC}"
    BLOCK_RESULT=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
         -H 'content-type:application/json;' http://localhost:${HTTP_PORT}/ext/bc/C/rpc)

    if echo "$BLOCK_RESULT" | grep -q "result"; then
        echo -e "${GREEN}✓ C-Chain RPC is responding!${NC}"
        echo -e "${BLUE}Block number: ${BLOCK_RESULT}${NC}"
    else
        echo -e "${YELLOW}⚠ C-Chain may still be initializing...${NC}"
    fi

    # Test balance query
    echo ""
    echo -e "${BLUE}Testing balance query for 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714...${NC}"
    BALANCE_RESULT=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x9011E888251AB053B7bD1cdB598Db4f9DEd94714","latest"],"id":1}' \
         -H 'content-type:application/json;' http://localhost:${HTTP_PORT}/ext/bc/C/rpc)

    if echo "$BALANCE_RESULT" | grep -q "result"; then
        echo -e "${GREEN}✓ Balance query successful!${NC}"
        echo -e "${BLUE}Balance: ${BALANCE_RESULT}${NC}"
    else
        echo -e "${YELLOW}⚠ Balance query may need more time...${NC}"
    fi

    echo ""
    echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}    LUX Clean POA Node Started Successfully!${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${BLUE}API Endpoints:${NC}"
    echo -e "  Info:     ${GREEN}http://localhost:${HTTP_PORT}/ext/info${NC}"
    echo -e "  Health:   ${GREEN}http://localhost:${HTTP_PORT}/ext/health${NC}"
    echo -e "  Metrics:  ${GREEN}http://localhost:${HTTP_PORT}/ext/metrics${NC}"
    echo -e "  C-Chain:  ${GREEN}http://localhost:${HTTP_PORT}/ext/bc/C/rpc${NC}"
    echo -e "  P-Chain:  ${GREEN}http://localhost:${HTTP_PORT}/ext/bc/P${NC}"
    echo -e "  X-Chain:  ${GREEN}http://localhost:${HTTP_PORT}/ext/bc/X${NC}"
    echo ""
    echo -e "${BLUE}Quick Tests:${NC}"
    echo -e "  Get block height:"
    echo -e "  ${YELLOW}curl -X POST --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}' -H 'content-type:application/json;' http://localhost:${HTTP_PORT}/ext/bc/C/rpc${NC}"
    echo ""
    echo -e "  Get balance for 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714:"
    echo -e "  ${YELLOW}curl -X POST --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"0x9011E888251AB053B7bD1cdB598Db4f9DEd94714\",\"latest\"],\"id\":1}' -H 'content-type:application/json;' http://localhost:${HTTP_PORT}/ext/bc/C/rpc${NC}"
    echo ""
    echo -e "${BLUE}Management:${NC}"
    echo -e "  View logs:    ${YELLOW}tail -f ${LOG_FILE}${NC}"
    echo -e "  Stop node:    ${YELLOW}kill ${PID}${NC}"
    echo ""
else
    echo -e "${RED}✗ Node failed to start!${NC}"
    echo -e "${YELLOW}Check the log file for details:${NC}"
    echo -e "${YELLOW}tail -100 ${LOG_FILE}${NC}"
    exit 1
fi
